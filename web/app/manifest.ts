import type { MetadataRoute } from "next";

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "Royal Move — Chess Analysis",
    short_name: "Royal Move",
    description: "Deep Stockfish analysis for Chess.com, Lichess, and PGN games.",
    start_url: "/",
    display: "standalone",
    background_color: "#faf8f2",
    theme_color: "#10231c",
  };
}
