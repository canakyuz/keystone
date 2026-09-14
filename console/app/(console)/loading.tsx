/** Shown while a console page waits on the API, in the shape of the page it replaces. */
export default function ConsoleLoading() {
  return (
    <div className="page" aria-busy="true">
      <p className="sr-only" role="status">
        Reading from the API
      </p>
      <span className="skeleton skeleton--title" aria-hidden="true" />
      <span className="skeleton skeleton--lede" aria-hidden="true" />
      <div className="kv block" aria-hidden="true">
        {[0, 1, 2, 3, 4].map((row) => (
          <div key={row} className="kv-row">
            <span className="skeleton" />
            <span className="skeleton" />
          </div>
        ))}
      </div>
    </div>
  );
}
