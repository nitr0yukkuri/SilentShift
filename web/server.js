const http = require("http");
const next = require("next");
const { Server } = require("socket.io");

const dev = process.env.NODE_ENV !== "production";
const host = "0.0.0.0";
const port = parseInt(process.env.PORT || "3000", 10);

const app = next({ dev, hostname: host, port });
const handle = app.getRequestHandler();

const rooms = new Map();

function ensureRoom(roomId) {
    if (!rooms.has(roomId)) {
        rooms.set(roomId, {
            players: new Map(),
            ball: { x: 0, y: 1.2, z: 0 }
        });
    }
    return rooms.get(roomId);
}

app.prepare().then(() => {
    const server = http.createServer((req, res) => handle(req, res));
    const io = new Server(server, {
        cors: {
            origin: "*"
        }
    });

    io.on("connection", (socket) => {
        socket.on("room:join", ({ roomId, name, color }) => {
            if (!roomId) {
                return;
            }

            socket.join(roomId);
            socket.data.roomId = roomId;
            socket.data.playerId = socket.id;

            const room = ensureRoom(roomId);
            room.players.set(socket.id, {
                id: socket.id,
                name: name || "Anonymous",
                color: color || "#ff9f1a",
                position: { x: 0, y: 0.5, z: 0 },
                score: 0
            });

            io.to(roomId).emit("state:update", {
                players: Array.from(room.players.values()),
                ball: room.ball
            });
        });

        socket.on("player:move", ({ position }) => {
            const roomId = socket.data.roomId;
            if (!roomId || !position) {
                return;
            }

            const room = ensureRoom(roomId);
            const player = room.players.get(socket.id);
            if (!player) {
                return;
            }

            player.position = position;
            socket.to(roomId).emit("player:move", { id: socket.id, position });
        });

        socket.on("ball:update", ({ ball }) => {
            const roomId = socket.data.roomId;
            if (!roomId || !ball) {
                return;
            }

            const room = ensureRoom(roomId);
            room.ball = ball;
            socket.to(roomId).emit("ball:update", { ball });
        });

        socket.on("goal", () => {
            const roomId = socket.data.roomId;
            if (!roomId) {
                return;
            }

            const room = ensureRoom(roomId);
            const player = room.players.get(socket.id);
            if (player) {
                player.score += 1;
            }

            io.to(roomId).emit("score:update", {
                players: Array.from(room.players.values())
            });
        });

        socket.on("disconnect", () => {
            const roomId = socket.data.roomId;
            if (!roomId) {
                return;
            }

            const room = ensureRoom(roomId);
            room.players.delete(socket.id);

            io.to(roomId).emit("state:update", {
                players: Array.from(room.players.values()),
                ball: room.ball
            });
        });
    });

    server.listen(port, host, () => {
        console.log(`SilentShift web running on http://${host}:${port}`);
    });
});
