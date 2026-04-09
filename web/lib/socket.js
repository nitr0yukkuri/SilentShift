import { io } from "socket.io-client";

let socket;

export function getSocket() {
    if (!socket) {
        socket = io({
            transports: ["websocket"],
            reconnection: true,
            reconnectionAttempts: 100,
            reconnectionDelay: 500
        });
    }

    return socket;
}
