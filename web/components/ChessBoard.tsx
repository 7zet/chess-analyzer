"use client";

const glyphs: Record<string, string> = {
  K: "♔",
  Q: "♕",
  R: "♖",
  B: "♗",
  N: "♘",
  P: "♙",
  k: "♚",
  q: "♛",
  r: "♜",
  b: "♝",
  n: "♞",
  p: "♟",
};

function squares(fen: string) {
  const board: string[] = [];
  const rows = (fen || "8/8/8/8/8/8/8/8").split(" ")[0].split("/");
  for (const row of rows) {
    for (const char of row) {
      const empty = Number(char);
      if (empty) for (let i = 0; i < empty; i++) board.push("");
      else board.push(char);
    }
  }
  return board;
}

export default function ChessBoard({ fen, lastMove }: { fen: string; lastMove?: string }) {
  const cells = squares(fen);
  const marked = new Set<string>();
  if (lastMove && lastMove.length >= 4) {
    marked.add(lastMove.slice(0, 2));
    marked.add(lastMove.slice(2, 4));
  }
  return (
    <div className="board-shell">
      <div className="board" role="img" aria-label="Chess position">
        {cells.map((piece, index) => {
          const file = "abcdefgh"[index % 8],
            rank = 8 - Math.floor(index / 8),
            name = file + rank;
          return (
            <div
              key={name}
              className={`square ${(index + Math.floor(index / 8)) % 2 ? "dark" : "light"} ${marked.has(name) ? "marked" : ""}`}
            >
              <span
                aria-hidden="true"
                className={piece === piece.toUpperCase() ? "white-piece" : "black-piece"}
              >
                {glyphs[piece]}
              </span>
              {index % 8 === 0 && <small className="rank">{rank}</small>}
              {rank === 1 && <small className="file">{file}</small>}
            </div>
          );
        })}
      </div>
    </div>
  );
}
