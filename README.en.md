# SilentShift (Silent Shift)

[日本語](README.md) | [English](README.en.md)

## 🚀 Concept

SilentShift is a cognitive-shift communication reactor that captures acoustic gaps in digital communication spaces—“the negative pressure of silence”—in real time with Go’s highly concurrent processing engine. It uses contextual analysis of conversation history through the Gemini API to introduce an “optimized absurdity” (a kusoge, or deliberately ridiculous game) at once, forcibly breaking social-entropy stagnation.

It replaces the subjective friction of “awkwardness” with a shared, non-verbal task (a kusoge), redirecting participants’ attention from “confrontation” to **“cooperation.”** It aims to transform relational tension caused by silence into emotional ease through laughter: a forced-intervention social accelerator for team building.

## 📋 Requirements

### Target and problem to solve

- **Problem:** Communication becomes strained because of awkward silences in voice chats such as Discord (“stagnant social entropy”).
- **Solution:** Detect silence in real time and forcibly guide users into a multiplayer 3D game that fits the preceding conversation but is just the right amount of absurd. Redirect attention from “confrontation” to “cooperation and laughter.”

### Functional requirements

1. **Voice-channel monitoring:** Continuously monitor Discord RTP packets and detect silence lasting for a configured duration (for example, 7 seconds).
2. **Context caching:** Temporarily keep recent text chat (or speech-recognition logs) in memory.
3. **AI context analysis:** When silence is detected, send the cached conversation history to the Gemini API, calculate an “awkwardness score” from the context, and dynamically select an “optimized kusoge theme.”
4. **Forced intervention:** Automatically post an AI-generated provocative message and a uniquely generated game-room (web app) URL to a Discord text channel.
5. **Multiplayer 3D space:** Users who open the URL can play a synchronized, real-time WebGL mini-game based on physics.

### Non-functional requirements

1. **Real-time behavior:** Keep audio-packet monitoring lightweight and minimize the delay from silence detection to posting a URL in Discord (with faster inference from Gemini 3.1 Flash).
2. **Scalability:** Safely manage Go routine lifecycles to support simultaneous operation across multiple servers and voice channels, preventing memory leaks.
3. **Availability:** Keep the bot server running continuously (CPU always allocated) rather than scaling it to zero, to maintain WebSocket connections.

## 🛠 Tech stack

### Backend (Bot & Core Logic)

- **Language:** Go
- **Library:** `bwmarrin/discordgo`
- **Characteristics:** Asynchronous packet monitoring with goroutines; thread-safe in-memory log management using a ring buffer and `sync.RWMutex`.

### AI context analysis

- **API:** Gemini 3.1 Flash API
- **Characteristics:** Fast “witty prompt / theme generation” using structured prompts that guide chain-of-thought and JSON output.

### Frontend (3D Kusoge Dimension)

- **Framework:** Next.js (App Router)
- **Rendering:** Three.js / React Three Fiber (R3F)
- **State synchronization:** Socket.io / Zustand
- **Characteristics:** Room generation through URL parameters and millisecond-level synchronization of physics coordinates in the browser.

## 🏗 System flow

1. **[Go]** Continuously monitor RTP packets from Discord voice channels.
2. **[Go]** Detect silence lasting N seconds.
3. **[Go → Gemini]** Send recent conversation logs to Gemini 3.1 Flash for contextual analysis.
4. **[Gemini → Go]** Return a context-appropriate kusoge theme and parameters as JSON.
5. **[Go → Discord]** Post an intervention message and web app (game room) URL to the channel.
6. **[Next.js + R3F]** Users open the URL and start real-time multiplayer in a WebGL-based 3D space.
