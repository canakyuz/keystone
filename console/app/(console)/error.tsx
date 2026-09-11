'use client';

export default function ConsoleError({ reset }: { error: Error; reset: () => void }) {
  return (
    <div className="page">
      <h1 className="page-title">The console could not read from the API</h1>
      <p className="page-lede">
        Check that the Keystone API is running at the address in KEYSTONE_API_URL, then try again. If it is running, its
        log has the reason.
      </p>
      <button className="button block" type="button" onClick={reset}>
        Try again
      </button>
    </div>
  );
}
