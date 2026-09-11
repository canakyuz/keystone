export function BlockHead({ id, title, note }: { id?: string; title: string; note?: React.ReactNode }) {
  return (
    <div className="block-head">
      <h2 id={id} className="block-title">
        {title}
      </h2>
      <span className="block-rule" aria-hidden="true" />
      {note ? <span className="block-note">{note}</span> : null}
    </div>
  );
}
