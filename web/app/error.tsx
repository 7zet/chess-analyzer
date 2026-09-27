"use client";

import { useEffect } from "react";

export default function ErrorPage({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error(error);
  }, [error]);
  return (
    <main className="error-page">
      <div className="loader-crest" aria-hidden="true">
        ♛
      </div>
      <p className="eyebrow">UNEXPECTED ERROR</p>
      <h1>The position could not be restored.</h1>
      <p>Please retry the last action. Your saved analyses remain safe.</p>
      <button className="primary" onClick={reset}>
        Try again
      </button>
    </main>
  );
}
