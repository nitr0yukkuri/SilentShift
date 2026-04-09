package discord

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"

	"silentshift/internal/ai"
	"silentshift/internal/config"
	"silentshift/internal/logcache"
)

type analyzer interface {
	Analyze(ctx context.Context, logs []logcache.Entry) (ai.Analysis, error)
}

type Bot struct {
	cfg      config.Config
	session  *discordgo.Session
	cache    *logcache.Cache
	analyzer analyzer

	mu       sync.Mutex
	watchers map[string]*silenceWatcher
}

type silenceWatcher struct {
	guildID          string
	voiceChannelID   string
	textChannelID    string
	lastPacketAt     time.Time
	lastIntervenedAt time.Time
	voiceConn        *discordgo.VoiceConnection
	cancel           context.CancelFunc
}

func NewBot(cfg config.Config, cache *logcache.Cache, analyzer analyzer) *Bot {
	return &Bot{
		cfg:      cfg,
		cache:    cache,
		analyzer: analyzer,
		watchers: make(map[string]*silenceWatcher),
	}
}

func (b *Bot) Start(ctx context.Context) error {
	s, err := discordgo.New("Bot " + b.cfg.DiscordToken)
	if err != nil {
		return err
	}

	s.Identify.Intents = discordgo.IntentsGuildMessages |
		discordgo.IntentsMessageContent |
		discordgo.IntentsGuildVoiceStates

	s.AddHandler(b.onMessageCreate)
	s.AddHandler(b.onVoiceStateUpdate)

	if err := s.Open(); err != nil {
		return err
	}

	b.session = s

	go func() {
		<-ctx.Done()
		_ = b.Close()
	}()

	log.Printf("SilentShift bot is online as %s", s.State.User.Username)
	return nil
}

func (b *Bot) Close() error {
	b.mu.Lock()
	for _, w := range b.watchers {
		w.cancel()
		if w.voiceConn != nil {
			_ = w.voiceConn.Disconnect()
		}
	}
	b.watchers = map[string]*silenceWatcher{}
	b.mu.Unlock()

	if b.session != nil {
		return b.session.Close()
	}
	return nil
}

func (b *Bot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil || m.Author.Bot {
		return
	}

	entry := logcache.Entry{
		Author:    m.Author.Username,
		Message:   m.Content,
		ChannelID: m.ChannelID,
		At:        time.Now().UTC(),
	}
	b.cache.Add(entry)

	content := strings.TrimSpace(m.Content)
	switch content {
	case "!silentshift join":
		b.handleJoinCommand(s, m)
	case "!silentshift leave":
		b.handleLeaveCommand(s, m)
	}
}

func (b *Bot) onVoiceStateUpdate(s *discordgo.Session, vs *discordgo.VoiceStateUpdate) {
	if s.State == nil || s.State.User == nil || vs.UserID == s.State.User.ID {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	w, ok := b.watchers[vs.GuildID]
	if !ok {
		return
	}

	if vs.ChannelID == w.voiceChannelID {
		w.lastPacketAt = time.Now()
	}
}

func (b *Bot) handleJoinCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	voiceChannelID := findUserVoiceChannel(s, m.GuildID, m.Author.ID)
	if voiceChannelID == "" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "先にVCへ入ってから `!silentshift join` を実行して。")
		return
	}

	b.mu.Lock()
	if old, ok := b.watchers[m.GuildID]; ok {
		old.cancel()
		if old.voiceConn != nil {
			_ = old.voiceConn.Disconnect()
		}
		delete(b.watchers, m.GuildID)
	}
	b.mu.Unlock()

	vc, err := s.ChannelVoiceJoin(m.GuildID, voiceChannelID, false, true)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, "VC接続に失敗: "+err.Error())
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	watcher := &silenceWatcher{
		guildID:        m.GuildID,
		voiceChannelID: voiceChannelID,
		textChannelID:  b.resolveTextChannel(m.ChannelID),
		lastPacketAt:   time.Now(),
		voiceConn:      vc,
		cancel:         cancel,
	}

	b.mu.Lock()
	b.watchers[m.GuildID] = watcher
	b.mu.Unlock()

	go b.monitorSilence(ctx, watcher)

	_, _ = s.ChannelMessageSend(m.ChannelID, "SilentShift起動: VC沈黙監視を開始。")
}

func (b *Bot) handleLeaveCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	b.mu.Lock()
	defer b.mu.Unlock()

	w, ok := b.watchers[m.GuildID]
	if !ok {
		_, _ = s.ChannelMessageSend(m.ChannelID, "監視は動いていません。")
		return
	}

	w.cancel()
	if w.voiceConn != nil {
		_ = w.voiceConn.Disconnect()
	}
	delete(b.watchers, m.GuildID)
	_, _ = s.ChannelMessageSend(m.ChannelID, "SilentShift停止: 監視を終了。")
}

func (b *Bot) monitorSilence(ctx context.Context, w *silenceWatcher) {
	ticker := time.NewTicker(400 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			if now.Sub(w.lastPacketAt) < b.cfg.SilenceThreshold {
				continue
			}
			if now.Sub(w.lastIntervenedAt) < b.cfg.InterveneCooldown {
				continue
			}

			logs := b.cache.LastN(24)
			analysis, err := b.analyzer.Analyze(ctx, logs)
			if err != nil {
				log.Printf("analysis failed: %v", err)
				continue
			}

			roomID := uuid.NewString()
			roomURL := buildRoomURL(b.cfg.RoomBaseURL, roomID, analysis.Theme, analysis.Params)

			message := fmt.Sprintf(
				"## SILENTSHIFT INTERVENTION\\n"+
					"%s\\n\\n"+
					"- awkwardness: **%d/100**\\n"+
					"- theme: **%s**\\n"+
					"- room: %s",
				analysis.TauntMessage,
				analysis.AwkwardnessScore,
				analysis.Theme,
				roomURL,
			)

			if _, err := b.session.ChannelMessageSend(w.textChannelID, message); err != nil {
				log.Printf("failed to post intervention: %v", err)
				continue
			}

			w.lastIntervenedAt = now
			w.lastPacketAt = now
		}
	}
}

func buildRoomURL(baseURL, roomID, theme string, params []string) string {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return fmt.Sprintf("%s/%s", strings.TrimRight(baseURL, "/"), roomID)
	}

	u.Path = strings.TrimRight(u.Path, "/") + "/" + roomID
	q := u.Query()
	q.Set("theme", theme)
	if len(params) > 0 {
		q.Set("params", strings.Join(params, ","))
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func (b *Bot) resolveTextChannel(defaultChannel string) string {
	if b.cfg.TargetTextChannel != "" {
		return b.cfg.TargetTextChannel
	}
	return defaultChannel
}

func findUserVoiceChannel(s *discordgo.Session, guildID, userID string) string {
	g, err := s.State.Guild(guildID)
	if err != nil || g == nil {
		return ""
	}

	for _, vs := range g.VoiceStates {
		if vs.UserID == userID {
			return vs.ChannelID
		}
	}

	return ""
}
