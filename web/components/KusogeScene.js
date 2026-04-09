"use client";

import { useEffect, useMemo, useRef } from "react";
import { Canvas, useFrame } from "@react-three/fiber";
import { Environment, OrbitControls, Text } from "@react-three/drei";
import { Physics, RigidBody } from "@react-three/rapier";
import { getSocket } from "../lib/socket";
import { useGameStore } from "../lib/store";

function PlayerRig({ me, color, socket, roomId }) {
    const ref = useRef();
    const keys = useRef({});
    const speed = 0.075;

    useEffect(() => {
        const down = (e) => {
            keys.current[e.key.toLowerCase()] = true;
        };
        const up = (e) => {
            keys.current[e.key.toLowerCase()] = false;
        };

        window.addEventListener("keydown", down);
        window.addEventListener("keyup", up);
        return () => {
            window.removeEventListener("keydown", down);
            window.removeEventListener("keyup", up);
        };
    }, []);

    useFrame(() => {
        if (!ref.current) {
            return;
        }

        const body = ref.current;
        const pos = body.translation();

        let x = pos.x;
        let z = pos.z;

        if (keys.current.w || keys.current.arrowup) z -= speed;
        if (keys.current.s || keys.current.arrowdown) z += speed;
        if (keys.current.a || keys.current.arrowleft) x -= speed;
        if (keys.current.d || keys.current.arrowright) x += speed;

        body.setTranslation({ x, y: 0.5, z }, true);
        socket.emit("player:move", { roomId, position: { x, y: 0.5, z } });
    });

    return (
        <RigidBody ref={ref} colliders="cuboid" restitution={0.2} friction={1.1}>
            <mesh position={[0, 0.5, 0]} castShadow>
                <capsuleGeometry args={[0.35, 0.9, 8, 16]} />
                <meshStandardMaterial color={color} metalness={0.3} roughness={0.2} />
            </mesh>
            <Text position={[0, 1.7, 0]} fontSize={0.3} color="#111">
                {me}
            </Text>
        </RigidBody>
    );
}

function BallSync({ socket, roomId }) {
    const ref = useRef();
    const updateBall = useGameStore((s) => s.updateBall);

    useFrame(() => {
        if (!ref.current) {
            return;
        }
        const p = ref.current.translation();
        const payload = { x: p.x, y: p.y, z: p.z };
        updateBall(payload);
        socket.emit("ball:update", { roomId, ball: payload });

        if (p.x > 8.5 && p.z > 8.5) {
            socket.emit("goal", { roomId });
            ref.current.setTranslation({ x: 0, y: 1.2, z: 0 }, true);
            ref.current.setLinvel({ x: 0, y: 0, z: 0 }, true);
        }
    });

    return (
        <RigidBody ref={ref} colliders="ball" restitution={0.85} friction={0.2}>
            <mesh castShadow>
                <icosahedronGeometry args={[0.9, 2]} />
                <meshStandardMaterial color="#f4d35e" metalness={0.8} roughness={0.1} />
            </mesh>
        </RigidBody>
    );
}

function RemotePlayers({ localId }) {
    const players = useGameStore((s) => s.players);

    return players
        .filter((p) => p.id !== localId)
        .map((p) => (
            <group key={p.id} position={[p.position.x, p.position.y, p.position.z]}>
                <mesh castShadow>
                    <capsuleGeometry args={[0.35, 0.9, 8, 16]} />
                    <meshStandardMaterial color={p.color} />
                </mesh>
                <Text position={[0, 1.7, 0]} fontSize={0.25} color="#222">
                    {p.name}
                </Text>
            </group>
        ));
}

function Overlay({ theme }) {
    const players = useGameStore((s) => s.players);
    return (
        <div className="hud">
            <div className="hud-title">{theme}</div>
            <div className="hud-list">
                {players.map((p) => (
                    <div key={p.id}>
                        {p.name}: {p.score}
                    </div>
                ))}
            </div>
            <div className="hud-help">Move: WASD / Arrow Keys</div>
        </div>
    );
}

export default function KusogeScene({ roomId, theme, identity }) {
    const socket = useMemo(() => getSocket(), []);
    const setStateSnapshot = useGameStore((s) => s.setStateSnapshot);
    const movePlayer = useGameStore((s) => s.movePlayer);
    const updateBall = useGameStore((s) => s.updateBall);
    const updateScore = useGameStore((s) => s.updateScore);

    useEffect(() => {
        socket.emit("room:join", {
            roomId,
            name: identity.name,
            color: identity.color
        });

        const onState = ({ players, ball }) => setStateSnapshot(players, ball);
        const onPlayerMove = ({ id, position }) => movePlayer(id, position);
        const onBall = ({ ball }) => updateBall(ball);
        const onScore = ({ players }) => updateScore(players);

        socket.on("state:update", onState);
        socket.on("player:move", onPlayerMove);
        socket.on("ball:update", onBall);
        socket.on("score:update", onScore);

        return () => {
            socket.off("state:update", onState);
            socket.off("player:move", onPlayerMove);
            socket.off("ball:update", onBall);
            socket.off("score:update", onScore);
        };
    }, [roomId, socket, identity, movePlayer, setStateSnapshot, updateBall, updateScore]);

    return (
        <div className="scene-wrap">
            <Overlay theme={theme} />
            <Canvas shadows camera={{ position: [10, 8, 10], fov: 50 }}>
                <color attach="background" args={["#fff6e0"]} />
                <ambientLight intensity={1.1} />
                <directionalLight position={[5, 8, 5]} castShadow intensity={1.8} />
                <Environment preset="sunset" />

                <Physics gravity={[0, -6.5, 0]}>
                    <RigidBody type="fixed" friction={1.2}>
                        <mesh receiveShadow position={[0, -0.1, 0]}>
                            <boxGeometry args={[24, 0.2, 24]} />
                            <meshStandardMaterial color="#9cc5a1" />
                        </mesh>
                    </RigidBody>

                    <RigidBody type="fixed" restitution={1.0}>
                        <mesh position={[9, 0.5, 9]}>
                            <boxGeometry args={[2.2, 1, 2.2]} />
                            <meshStandardMaterial color="#e76f51" />
                        </mesh>
                    </RigidBody>

                    <PlayerRig
                        me={identity.name}
                        color={identity.color}
                        roomId={roomId}
                        socket={socket}
                    />
                    <BallSync roomId={roomId} socket={socket} />
                </Physics>

                <RemotePlayers localId={socket.id} />
                <OrbitControls enablePan={false} minDistance={8} maxDistance={28} />
            </Canvas>
        </div>
    );
}
