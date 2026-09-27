import type { Metadata } from "next";
import "./styles.css";

export const metadata: Metadata = {
  metadataBase: new URL("https://royalmove.local"),
  title: { default: "Royal Move — Chess Analysis", template: "%s · Royal Move" },
  description: "Deep Stockfish analysis for Chess.com, Lichess, and PGN games.",
  applicationName: "Royal Move",
  keywords: ["chess analysis", "Stockfish", "PGN", "Chess.com", "Lichess"],
  authors: [{ name: "Royal Move" }],
  openGraph: {
    title: "Royal Move — Chess Analysis",
    description: "Deep, objective analysis for every move.",
    type: "website",
    siteName: "Royal Move",
  },
  robots: { index: true, follow: true },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
