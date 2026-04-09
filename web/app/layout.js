import "./globals.css";

export const metadata = {
    title: "SilentShift",
    description: "Silence Intervention Multiplayer Kusoge"
};

export default function RootLayout({ children }) {
    return (
        <html lang="ja">
            <body>{children}</body>
        </html>
    );
}
