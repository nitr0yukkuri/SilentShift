"use client";

import { useMemo } from "react";
import { useParams, useSearchParams } from "next/navigation";
import KusogeScene from "../../../components/KusogeScene";

const COLORS = ["#ff6f61", "#0db39e", "#f4d35e", "#1f7a8c", "#e36414"];

export default function RoomPage() {
    const params = useParams();
    const searchParams = useSearchParams();
    const roomId = params.roomId;
    const theme = searchParams.get("theme") || "無題の不条理チャレンジ";

    const identity = useMemo(() => {
        const color = COLORS[Math.floor(Math.random() * COLORS.length)];
        const name = `Player-${Math.floor(Math.random() * 900 + 100)}`;
        return { color, name };
    }, []);

    return (
        <main className="room-root">
            <header className="room-header">
                <div>
                    <h1>SilentShift Room</h1>
                    <p>{theme}</p>
                </div>
                <div className="chip">Room: {roomId}</div>
            </header>
            <section className="game-wrap">
                <KusogeScene roomId={roomId} theme={theme} identity={identity} />
            </section>
        </main>
    );
}
