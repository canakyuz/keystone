// Written by scripts/plates.py. Edit the script, not this file.

export const PLATES = {
  vault: { src: "/plates/vault.svg", width: 417, height: 614, alt: "Engraving of a Roman barrel vault laid stone by stone over its timber centering, a mason standing below", credit: "Viollet-le-Duc, Dictionnaire raisonné de l'architecture, 1856", source: "https://commons.wikimedia.org/wiki/File:Construction.voute.romaine.png" },
  pier: { src: "/plates/pier.svg", width: 320, height: 318, alt: "Engraving of two arches springing from a single pier", credit: "Viollet-le-Duc, Dictionnaire raisonné de l'architecture, 1856", source: "https://commons.wikimedia.org/wiki/File:Tas.de.charge.2.png" },
  masons: { src: "/plates/masons.svg", width: 604, height: 288, alt: "Engraving of masons cutting and setting stone in a yard before a brick arch", credit: "Diderot and d'Alembert, Encyclopédie, Maçonnerie plate I, 1762", source: "https://commons.wikimedia.org/wiki/File:Engraving_from_Diderot%2C_Encyclop%C3%A9die%2C_v._1%2C_pl._194%2C_Architecture_Maconnerie._Masonry_arch._LCCN2006677828.jpg" },
} as const;

export type PlateName = keyof typeof PLATES;
