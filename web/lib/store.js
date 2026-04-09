import { create } from "zustand";

export const useGameStore = create((set) => ({
    players: [],
    ball: { x: 0, y: 1.2, z: 0 },
    setStateSnapshot: (players, ball) => set({ players, ball }),
    movePlayer: (id, position) =>
        set((s) => ({
            players: s.players.map((p) => (p.id === id ? { ...p, position } : p))
        })),
    updateBall: (ball) => set({ ball }),
    updateScore: (players) => set({ players })
}));
