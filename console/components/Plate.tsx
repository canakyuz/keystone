import { PLATES, type PlateName } from '@/lib/plates';

/**
 * An engraving from lib/plates.ts, cut by scripts/plates.py.
 *
 * The SVG is a mask over currentColor rather than an <img>, so the ink is the page's own ink
 * and the plate holds in both themes without a second file. The box takes the plate's aspect
 * ratio, so nothing shifts while the mask loads.
 */
export function Plate({ name, label, className }: { name: PlateName; label: string; className?: string }) {
  const plate = PLATES[name];
  const mask = `url(${plate.src})`;

  return (
    <figure className={className ? `plate ${className}` : 'plate'}>
      <span
        className="plate-ink"
        role="img"
        aria-label={plate.alt}
        style={{ aspectRatio: `${plate.width} / ${plate.height}`, maskImage: mask, WebkitMaskImage: mask }}
      />
      <figcaption className="plate-caption">
        <span>{label}</span>
        <span className="plate-credit">{plate.credit}</span>
      </figcaption>
    </figure>
  );
}
