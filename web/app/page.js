"use client";

import { useMemo } from "react";
import { useRouter } from "next/navigation";
import { v4 as uuidv4 } from "uuid";

export default function Home() {
    const router = useRouter();
    const seedTheme = useMemo(() => "段差の向こうの友情バトル", []);

    const createRoom = () => {
        const roomId = uuidv4();
        router.push(`/room/${roomId}?theme=${encodeURIComponent(seedTheme)}`);
    };

    return (
        <main className="landing">
            <div className="noise" />
            <section className="hero">
                <h1>
                    SILENT
                    <span>SHIFT</span>
                </h1>
                <p>
                    沈黙を検知したら、対話は中断。<br />
                    いまから始まるのは、共闘でしか突破できない不条理。
                </p>
                <button type="button" onClick={createRoom}>
                    緊急ルームを生成
                </button>
            </section>
        </main>
    );
}
