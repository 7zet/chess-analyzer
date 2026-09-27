export const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export type Source = "pgn" | "chesscom" | "lichess";
export type AnalysisStatus = "queued" | "processing" | "complete" | "failed";

export type User = {
  id: string;
  email: string;
  displayName: string;
  role: "user" | "admin";
  status: "active" | "blocked";
  createdAt: string;
};
export type GamePreview = {
  id: string;
  source: Source;
  white: string;
  black: string;
  whiteElo?: number;
  blackElo?: number;
  result: string;
  playedAt: string;
  timeClass?: string;
  opening?: string;
  pgn: string;
};
export type MoveAnalysis = {
  ply: number;
  san: string;
  uci: string;
  fenBefore: string;
  fenAfter: string;
  scoreBefore?: number;
  scoreAfter?: number;
  mateBefore?: number;
  mateAfter?: number;
  bestMove: string;
  principalVariation: string[];
  classification: string;
  loss: number;
  explanation: string;
};
export type Analysis = {
  id: string;
  source: Source;
  whiteName: string;
  blackName: string;
  result: string;
  pgn: string;
  status: AnalysisStatus;
  progress: number;
  error?: string;
  summary?: {
    accuracy: number;
    averageLoss: number;
    totalMoves: number;
    classifications: Record<string, number>;
  };
  createdAt: string;
  moves?: MoveAnalysis[];
};

export class ApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export async function request<T>(path: string, init: RequestInit = {}, token?: string): Promise<T> {
  let response: Response;
  try {
    response = await fetch(API + path, {
      ...init,
      cache: "no-store",
      signal: init.signal ?? AbortSignal.timeout(40_000),
      headers: {
        "Content-Type": "application/json",
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...init.headers,
      },
    });
  } catch (error) {
    if (error instanceof DOMException && error.name === "TimeoutError") {
      throw new ApiError("The request timed out. Please try again.", 408);
    }
    throw new ApiError("Royal Move could not reach the analysis service.", 0);
  }
  const data = await response
    .json()
    .catch(() => ({ error: "The server response could not be read" }));
  if (!response.ok) {
    throw new ApiError(data.error || "The request could not be completed", response.status);
  }
  return data as T;
}
