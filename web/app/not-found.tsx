import Link from "next/link";

export default function NotFound() {
  return (
    <main className="error-page">
      <div className="loader-crest" aria-hidden="true">
        ♚
      </div>
      <p className="eyebrow">404 · OFF THE BOARD</p>
      <h1>This page does not exist.</h1>
      <Link className="primary" href="/">
        Return home
      </Link>
    </main>
  );
}
