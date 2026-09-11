import Link from 'next/link';

export default function ConsoleNotFound() {
  return (
    <div className="page">
      <h1 className="page-title">Nothing here</h1>
      <p className="page-lede">
        Either it does not exist, or it belongs to someone else. The API gives the same answer for both, on purpose.
      </p>
      <p className="block">
        <Link href="/" className="link">
          Back to your tenant
        </Link>
      </p>
    </div>
  );
}
