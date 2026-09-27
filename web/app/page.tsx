"use client";

import { FormEvent, useEffect, useState } from "react";
import ChessBoard from "@/components/ChessBoard";
import AuthDialog from "@/components/AuthDialog";
import { Analysis, GamePreview, MoveAnalysis, request, User } from "@/lib/api";

type View = "home" | "workspace" | "history" | "admin";

const START_FEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1";
const labels: Record<string, { name: string; symbol: string }> = {
  precise: { name: "Precise", symbol: "◆" },
  strong: { name: "Strong", symbol: "●" },
  good: { name: "Good", symbol: "✓" },
  acceptable: { name: "Acceptable", symbol: "○" },
  dubious: { name: "Dubious", symbol: "?!" },
  mistake: { name: "Mistake", symbol: "×" },
  critical: { name: "Critical Error", symbol: "!!" },
};

export default function Home() {
  const [token, setToken] = useState("");
  const [user, setUser] = useState<User | null>(null);
  const [authOpen, setAuthOpen] = useState(false);
  const [view, setView] = useState<View>("home");
  const [history, setHistory] = useState<Analysis[]>([]);
  const [active, setActive] = useState<Analysis | null>(null);
  useEffect(() => {
    const saved = localStorage.getItem("royal-token");
    if (saved) setToken(saved);
  }, []);
  useEffect(() => {
    if (!token) return;
    Promise.all([
      request<User>("/api/me", {}, token),
      request<{ analyses: Analysis[] }>("/api/analyses", {}, token),
    ])
      .then(([currentUser, data]) => {
        setUser(currentUser);
        setHistory(data.analyses);
        localStorage.setItem("royal-user", JSON.stringify(currentUser));
      })
      .catch(() => {
        localStorage.removeItem("royal-token");
        localStorage.removeItem("royal-user");
        setToken("");
        setUser(null);
      });
  }, [token]);
  function authenticated(next: () => void) {
    if (!token) {
      setAuthOpen(true);
      return;
    }
    next();
  }
  function loggedIn(nextToken: string, nextUser: User) {
    localStorage.setItem("royal-token", nextToken);
    localStorage.setItem("royal-user", JSON.stringify(nextUser));
    setToken(nextToken);
    setUser(nextUser);
    setAuthOpen(false);
    setView("workspace");
  }
  function logout() {
    localStorage.removeItem("royal-token");
    localStorage.removeItem("royal-user");
    setToken("");
    setUser(null);
    setHistory([]);
    setView("home");
  }
  function openAnalysis(item: Analysis) {
    setActive(item);
    setView("workspace");
  }
  return (
    <>
      <a className="skip-link" href="#main-content">
        Skip to main content
      </a>
      <main id="main-content">
        <Header
          user={user}
          view={view}
          onView={(v) => authenticated(() => setView(v))}
          onLogin={() => setAuthOpen(true)}
          onLogout={logout}
        />
        {view === "home" && <Landing onStart={() => authenticated(() => setView("workspace"))} />}
        {view === "workspace" && (
          <Workspace
            token={token}
            active={active}
            setActive={setActive}
            onCreated={(item) => {
              setHistory((h) => [item, ...h]);
              setActive(item);
            }}
          />
        )}
        {view === "history" && <History items={history} open={openAnalysis} />}
        {view === "admin" && user?.role === "admin" && <Admin items={history} token={token} />}
        <Footer />
        {authOpen && <AuthDialog onClose={() => setAuthOpen(false)} onSuccess={loggedIn} />}
      </main>
    </>
  );
}

function Header({
  user,
  view,
  onView,
  onLogin,
  onLogout,
}: {
  user: User | null;
  view: View;
  onView: (v: View) => void;
  onLogin: () => void;
  onLogout: () => void;
}) {
  return (
    <header className="site-header">
      <button className="brand" onClick={() => onView("home")}>
        <span className="brand-mark">♛</span>
        <span>
          <b>ROYAL MOVE</b>
          <small>CHESS ANALYSIS</small>
        </span>
      </button>
      <nav>
        <button
          className={view === "workspace" ? "active" : ""}
          onClick={() => onView("workspace")}
        >
          Analysis
        </button>
        <button className={view === "history" ? "active" : ""} onClick={() => onView("history")}>
          History
        </button>
        {user?.role === "admin" && (
          <button className={view === "admin" ? "active" : ""} onClick={() => onView("admin")}>
            Administration
          </button>
        )}
      </nav>
      {user ? (
        <div className="profile">
          <span className="avatar">{user.displayName?.[0]?.toUpperCase()}</span>
          <span>
            <b>{user.displayName}</b>
            <small>{user.role === "admin" ? "Administrator" : "Member"}</small>
          </span>
          <button onClick={onLogout} title="Sign out">
            ↗
          </button>
        </div>
      ) : (
        <button className="outline" onClick={onLogin}>
          Sign in
        </button>
      )}
    </header>
  );
}

function Landing({ onStart }: { onStart: () => void }) {
  return (
    <>
      <section className="hero">
        <div className="hero-copy">
          <p className="eyebrow">PRECISION. DISCIPLINE. MASTERY.</p>
          <h1>
            Know the true value
            <br />
            of <em>every move.</em>
          </h1>
          <p className="lead">
            Run your Chess.com, Lichess, or PGN games through deep Stockfish analysis. Understand
            every decision with calm, precise, and practical insight.
          </p>
          <div className="hero-actions">
            <button className="primary" onClick={onStart}>
              Analyze a game <span>→</span>
            </button>
            <span className="quiet">
              <i>♞</i> Professional-grade analysis
            </span>
          </div>
          <div className="trust-row">
            <div>
              <b>3</b>
              <span>analysis sources</span>
            </div>
            <div>
              <b>100%</b>
              <span>objective evaluation</span>
            </div>
            <div>
              <b>24/7</b>
              <span>analysis queue</span>
            </div>
          </div>
        </div>
        <div className="hero-board">
          <div className="ornament top">✦</div>
          <ChessBoard fen="r1bq1rk1/ppp2ppp/2n2n2/3pp3/1b1PP3/2N2N2/PPP1BPPP/R1BQ1RK1 w - - 4 8" />
          <div className="position-card">
            <span className="gold-dot"></span>
            <div>
              <small>POSITION EVALUATION</small>
              <b>White is slightly better</b>
            </div>
            <strong>+0.62</strong>
          </div>
        </div>
      </section>
      <section className="principles">
        <p className="eyebrow center">THREE PATHS. ONE DEFINITIVE ANALYSIS.</p>
        <h2>Your games, wherever you play</h2>
        <div className="principle-grid">
          <article>
            <span>01</span>
            <div className="line-icon">♟</div>
            <h3>Chess.com</h3>
            <p>Enter a username and choose from the player’s most recent games.</p>
          </article>
          <article>
            <span>02</span>
            <div className="line-icon">♜</div>
            <h3>Lichess</h3>
            <p>Import public profile games instantly and begin a complete analysis.</p>
          </article>
          <article>
            <span>03</span>
            <div className="line-icon">♝</div>
            <h3>PGN notation</h3>
            <p>Paste the moves directly. Full headers and a final result are optional.</p>
          </article>
        </div>
      </section>
    </>
  );
}

function Workspace({
  token,
  active,
  setActive,
  onCreated,
}: {
  token: string;
  active: Analysis | null;
  setActive: (a: Analysis | null) => void;
  onCreated: (a: Analysis) => void;
}) {
  return (
    <section className="workspace-wrap">
      {active ? (
        <AnalysisRoom
          token={token}
          analysis={active}
          setAnalysis={setActive}
          onBack={() => setActive(null)}
        />
      ) : (
        <ImportPanel token={token} onCreated={onCreated} />
      )}
    </section>
  );
}

function ImportPanel({ token, onCreated }: { token: string; onCreated: (a: Analysis) => void }) {
  const [tab, setTab] = useState<"chesscom" | "lichess" | "pgn">("chesscom");
  const [username, setUsername] = useState("");
  const [pgn, setPgn] = useState("1. e4 e5\n2. Nf3 Nc6\n3. Bb5 a6\n4. Ba4 Nf6\n5. d3");
  const [games, setGames] = useState<GamePreview[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function find(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const data = await request<{ games: GamePreview[] }>(
        `/api/platforms/${tab}/${encodeURIComponent(username)}/games`,
      );
      setGames(data.games);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Something went wrong");
    } finally {
      setBusy(false);
    }
  }
  async function create(source: string, text: string, id?: string) {
    setBusy(true);
    setError("");
    try {
      const item = await request<Analysis>(
        "/api/analyses",
        { method: "POST", body: JSON.stringify({ source, pgn: text, externalId: id || "" }) },
        token,
      );
      onCreated(item);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Something went wrong");
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="import-layout">
      <aside className="import-intro">
        <p className="eyebrow">NEW ANALYSIS</p>
        <h1>Select a game</h1>
        <p>Choose a source. We will handle every calculation from there.</p>
        <div className="engine-seal">
          <span>♚</span>
          <div>
            <small>ANALYSIS ENGINE</small>
            <b>Stockfish · Deep mode</b>
          </div>
        </div>
      </aside>
      <div className="import-card">
        <div className="tabs">
          <button
            className={tab === "chesscom" ? "active" : ""}
            onClick={() => {
              setTab("chesscom");
              setGames([]);
            }}
          >
            Chess.com
          </button>
          <button
            className={tab === "lichess" ? "active" : ""}
            onClick={() => {
              setTab("lichess");
              setGames([]);
            }}
          >
            Lichess
          </button>
          <button className={tab === "pgn" ? "active" : ""} onClick={() => setTab("pgn")}>
            PGN
          </button>
        </div>
        {tab === "pgn" ? (
          <div className="pgn-form">
            <label>GAME NOTATION</label>
            <textarea value={pgn} onChange={(e) => setPgn(e.target.value)} spellCheck={false} />
            <div className="form-note">
              <span>ℹ</span> Standard move notation and complete PGN records are accepted.
            </div>
            <button className="primary wide" onClick={() => create("pgn", pgn)} disabled={busy}>
              {busy ? "VALIDATING…" : "START ANALYSIS →"}
            </button>
          </div>
        ) : (
          <>
            <form className="username-form" onSubmit={find}>
              <label>{tab === "chesscom" ? "CHESS.COM" : "LICHESS"} USERNAME</label>
              <div>
                <input
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder="Example: DrNykterstein"
                  required
                />
                <button className="primary" disabled={busy}>
                  {busy ? "SEARCHING…" : "FIND GAMES"}
                </button>
              </div>
            </form>
            {games.length > 0 && (
              <div className="game-list">
                <div className="list-head">
                  <b>Recent games</b>
                  <span>{games.length} found</span>
                </div>
                {games.map((game) => (
                  <button
                    className="game-row"
                    key={game.id}
                    onClick={() => create(game.source, game.pgn, game.id)}
                  >
                    <div className="game-piece">♞</div>
                    <div className="players">
                      <b>
                        {game.white} <small>{game.whiteElo}</small>
                      </b>
                      <span>
                        {game.black} <small>{game.blackElo}</small>
                      </span>
                    </div>
                    <strong>{game.result}</strong>
                    <div className="game-meta">
                      <span>{game.timeClass}</span>
                      <small>{new Date(game.playedAt).toLocaleDateString("en-GB")}</small>
                    </div>
                    <i>→</i>
                  </button>
                ))}
              </div>
            )}
          </>
        )}
        {error && <p className="form-error pad">{error}</p>}
      </div>
    </div>
  );
}

function AnalysisRoom({
  token,
  analysis,
  setAnalysis,
  onBack,
}: {
  token: string;
  analysis: Analysis;
  setAnalysis: (a: Analysis) => void;
  onBack: () => void;
}) {
  const [selected, setSelected] = useState(0);
  const [retrying, setRetrying] = useState(false);
  const [actionError, setActionError] = useState("");
  const ready = analysis.status === "complete" && Boolean(analysis.moves?.length);

  async function retryAnalysis() {
    setRetrying(true);
    setActionError("");
    try {
      await request(`/api/analyses/${analysis.id}/retry`, { method: "POST" }, token);
      setAnalysis({
        ...analysis,
        status: "queued",
        progress: 0,
        error: undefined,
        moves: undefined,
      });
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "The analysis could not be retried.");
    } finally {
      setRetrying(false);
    }
  }
  useEffect(() => {
    if (ready || analysis.status === "failed") return;
    let mounted = true;
    const refresh = async () => {
      try {
        const fresh = await request<Analysis>(`/api/analyses/${analysis.id}`, {}, token);
        if (mounted) setAnalysis(fresh);
      } catch {}
    };
    refresh();
    const timer = setInterval(refresh, 1500);
    return () => {
      mounted = false;
      clearInterval(timer);
    };
  }, [analysis.id, analysis.status, ready, token, setAnalysis]);
  if (!ready)
    return (
      <div className="analysis-loading">
        <button className="back" onClick={onBack}>
          ← Back
        </button>
        <div className="loader-crest">♛</div>
        <p className="eyebrow">
          {analysis.status === "failed" ? "ANALYSIS STOPPED" : "DEEP ANALYSIS"}
        </p>
        <h2>
          {analysis.status === "failed"
            ? analysis.error
            : analysis.status === "complete"
              ? "Loading analysis results"
              : "Examining every position"}
        </h2>
        {analysis.status !== "failed" && (
          <>
            <div className="progress">
              <i style={{ width: `${analysis.progress}%` }} />
            </div>
            <b>{analysis.progress}%</b>
            <p>Stockfish is calculating the strongest alternatives for every move.</p>
          </>
        )}
        {analysis.status === "failed" && (
          <>
            {actionError && (
              <p className="form-error" role="alert">
                {actionError}
              </p>
            )}
            <button className="primary" onClick={retryAnalysis} disabled={retrying}>
              {retrying ? "RETRYING…" : "RETRY ANALYSIS"}
            </button>
          </>
        )}
      </div>
    );
  const moves = analysis.moves || [];
  const move = moves[selected];
  const fen = move?.fenAfter || START_FEN;
  return (
    <div className="analysis-page">
      <div className="analysis-top">
        <button className="back" onClick={onBack}>
          ← New analysis
        </button>
        <div>
          <h1>
            {analysis.whiteName} <span>—</span> {analysis.blackName}
          </h1>
          <p>
            {analysis.result} · {analysis.source.toUpperCase()}
          </p>
        </div>
        <div className="accuracy">
          <small>ACCURACY</small>
          <b>{analysis.summary?.accuracy ?? 0}%</b>
        </div>
      </div>
      <div className="analysis-grid">
        <div>
          <ChessBoard fen={fen} lastMove={move?.uci} />
          <div className="board-controls">
            <button aria-label="First move" onClick={() => setSelected(0)}>
              «
            </button>
            <button
              aria-label="Previous move"
              onClick={() => setSelected(Math.max(0, selected - 1))}
            >
              ‹
            </button>
            <span>
              {Math.floor(selected / 2) + 1}. {move?.san}
            </span>
            <button
              aria-label="Next move"
              onClick={() => setSelected(Math.min(moves.length - 1, selected + 1))}
            >
              ›
            </button>
            <button aria-label="Last move" onClick={() => setSelected(moves.length - 1)}>
              »
            </button>
          </div>
        </div>
        <section className="move-panel">
          <div className="eval-header">
            <span className={`class-badge ${move?.classification}`}>
              {labels[move?.classification]?.symbol}
            </span>
            <div>
              <small>
                {selected % 2 === 0 ? "WHITE" : "BLACK"} · MOVE {Math.floor(selected / 2) + 1}
              </small>
              <h2>
                {move?.san} <em>{labels[move?.classification]?.name}</em>
              </h2>
            </div>
            <strong>{formatScore(move?.scoreAfter, move?.mateAfter)}</strong>
          </div>
          <p className="explanation">{move?.explanation}</p>
          <div className="move-details">
            <div>
              <small>BEST MOVE</small>
              <b>{move?.bestMove || "—"}</b>
            </div>
            <div>
              <small>EVALUATION LOSS</small>
              <b>{move?.loss ? `${(move.loss / 100).toFixed(2)}` : "0.00"}</b>
            </div>
          </div>
          <div className="pv">
            <small>PRINCIPAL VARIATION</small>
            <p>{move?.principalVariation?.slice(0, 8).join("  ") || "—"}</p>
          </div>
          <div className="moves-scroll">
            {pairMoves(moves).map((pair, i) => (
              <div key={i}>
                <span>{i + 1}.</span>
                {pair.map((m) => (
                  <button
                    key={m.ply}
                    className={`${selected === m.ply - 1 ? "selected" : ""} ${m.classification}`}
                    onClick={() => setSelected(m.ply - 1)}
                  >
                    {m.san}
                    <i>{labels[m.classification]?.symbol}</i>
                  </button>
                ))}
              </div>
            ))}
          </div>
        </section>
      </div>
    </div>
  );
}

function History({ items, open }: { items: Analysis[]; open: (a: Analysis) => void }) {
  return (
    <section className="page-section">
      <p className="eyebrow">PERSONAL LIBRARY</p>
      <h1>Analysis history</h1>
      <p className="section-lead">Every game is a lesson for your next decisive move.</p>
      <div className="history-table">
        <div className="history-head">
          <span>GAME</span>
          <span>SOURCE</span>
          <span>STATUS</span>
          <span>DATE</span>
          <span></span>
        </div>
        {items.length ? (
          items.map((item) => (
            <button key={item.id} onClick={() => open(item)}>
              <span>
                <i>♟</i>
                <b>
                  {item.whiteName} — {item.blackName}
                </b>
                <small>{item.result}</small>
              </span>
              <span>{item.source}</span>
              <span className={`status ${item.status}`}>{statusName(item.status)}</span>
              <span>{new Date(item.createdAt).toLocaleDateString("en-GB")}</span>
              <strong>→</strong>
            </button>
          ))
        ) : (
          <div className="empty">No analyses yet.</div>
        )}
      </div>
    </section>
  );
}

function Admin({ items, token }: { items: Analysis[]; token: string }) {
  const [users, setUsers] = useState<User[]>([]);
  const [userError, setUserError] = useState("");
  const complete = items.filter((i) => i.status === "complete").length,
    failed = items.filter((i) => i.status === "failed").length,
    active = items.filter((i) => i.status === "processing" || i.status === "queued").length;

  useEffect(() => {
    request<{ users: User[] }>("/api/admin/users", {}, token)
      .then((data) => setUsers(data.users))
      .catch((error) =>
        setUserError(error instanceof Error ? error.message : "Users could not be loaded."),
      );
  }, [token]);

  async function toggleUser(target: User) {
    const status = target.status === "active" ? "blocked" : "active";
    setUserError("");
    try {
      await request(
        `/api/admin/users/${target.id}/status`,
        { method: "PATCH", body: JSON.stringify({ status }) },
        token,
      );
      setUsers((current) =>
        current.map((user) => (user.id === target.id ? { ...user, status } : user)),
      );
    } catch (error) {
      setUserError(error instanceof Error ? error.message : "The account could not be updated.");
    }
  }
  return (
    <section className="page-section">
      <p className="eyebrow">CONTROL CENTRE</p>
      <h1>System overview</h1>
      <div className="admin-stats">
        <article>
          <small>TOTAL ANALYSES</small>
          <b>{items.length}</b>
          <span>All time</span>
        </article>
        <article>
          <small>COMPLETED</small>
          <b>{complete}</b>
          <span>Reports ready</span>
        </article>
        <article>
          <small>IN QUEUE</small>
          <b>{active}</b>
          <span>Active processes</span>
        </article>
        <article>
          <small>FAILED</small>
          <b>{failed}</b>
          <span>Requires attention</span>
        </article>
      </div>
      <div className="admin-card">
        <h2>Recent jobs</h2>
        {items.slice(0, 10).map((item) => (
          <div className="admin-row" key={item.id}>
            <code>{item.id.slice(0, 8)}</code>
            <span>
              {item.whiteName} — {item.blackName}
            </span>
            <span>{item.source}</span>
            <i className={`status ${item.status}`}>{statusName(item.status)}</i>
            <small>{item.progress}%</small>
          </div>
        ))}
      </div>
      <div className="admin-card">
        <h2>Members</h2>
        {userError && (
          <p className="form-error" role="alert">
            {userError}
          </p>
        )}
        {users.map((member) => (
          <div className="member-row" key={member.id}>
            <span className="avatar">{member.displayName[0]?.toUpperCase()}</span>
            <span>
              <b>{member.displayName}</b>
              <small>{member.email}</small>
            </span>
            <i className={`status ${member.status}`}>{member.status}</i>
            <span>{member.role}</span>
            <button
              className="outline compact"
              onClick={() => toggleUser(member)}
              disabled={member.role === "admin"}
            >
              {member.status === "active" ? "Block" : "Restore"}
            </button>
          </div>
        ))}
      </div>
    </section>
  );
}

function Footer() {
  return (
    <footer>
      <div className="brand">
        <span className="brand-mark">♛</span>
        <span>
          <b>ROYAL MOVE</b>
          <small>CHESS ANALYSIS</small>
        </span>
      </div>
      <p>Created for a deeper understanding of chess.</p>
      <span>© {new Date().getFullYear()} Royal Move</span>
    </footer>
  );
}
function pairMoves(moves: MoveAnalysis[]) {
  const pairs: MoveAnalysis[][] = [];
  moves.forEach((m, i) => {
    if (i % 2 === 0) pairs.push([m]);
    else pairs[pairs.length - 1].push(m);
  });
  return pairs;
}
function formatScore(score?: number, mate?: number) {
  if (mate !== undefined) return `M${Math.abs(mate)}`;
  if (score === undefined) return "—";
  return `${score >= 0 ? "+" : ""}${(score / 100).toFixed(2)}`;
}
function statusName(status: string) {
  const statuses: Record<string, string> = {
    queued: "Queued",
    processing: "Analyzing",
    complete: "Complete",
    failed: "Failed",
  };
  return statuses[status] || status;
}
