#!/usr/bin/env python3
"""Fail if a code comment is not written in English.

Why this exists as a script rather than a habit: this repository was written in Turkish
first and translated later. Two hand-rolled sweeps both reported "clean" and both were
wrong. A grep for Turkish characters missed every comment written without diacritics,
and a hand-written word list only ever catches the words somebody remembered to list.

So the guard is built the other way round. A small vendored list of Turkish function
words and suffixes runs everywhere and is what gates CI; it is deterministic and needs
no system packages. When an English dictionary happens to be installed, a second,
stricter pass also reports comment tokens that are in no dictionary at all, which is how
the remaining Turkish was found in the first place.

    python3 scripts/langcheck.py            # gate: Turkish markers
    python3 scripts/langcheck.py --strict   # also run the dictionary pass, when available
"""
import collections
import os
import re
import subprocess
import sys

# Turkish function words and inflected forms. Chosen so that no entry is also an English
# word: a false positive here fails the build, which is worse than missing one token.
TURKISH_MARKERS = {
    "acikca", "acilir", "adet", "adimlar", "adimlari", "ait", "aksi", "alinir", "alir",
    "almaz", "ancak", "anlamina", "anlatir", "aracı", "araligi", "arasinda", "artik",
    "ayni", "ayri", "ayrica", "ayrim", "bagimli", "bagimlilik", "baglanti", "bakar",
    "baska", "baslar", "basari", "basarili", "basarisiz", "belirler", "bilgisi",
    "bildirir", "bilincli", "birden", "birlikte", "boyle", "boylece", "bunlar", "bunun",
    "buyuk", "buyur", "cagirir", "cagrilir", "calisan", "calisir", "calistirir", "cikar",
    "cunku", "daha", "davranis", "degeri", "degil", "degildir", "degisir", "demektir",
    "devam", "dolayi", "dolayisiyla", "dondurur", "doner", "duser", "edilir", "eder",
    "etmez", "gecerli", "gecis", "gecmis", "gelen", "gelir", "gerekir", "gerekli",
    "gerekce", "geri", "gibi", "gorunur", "hala", "halinde", "hangi", "hata", "hatali",
    "hatasi", "henuz", "hicbir", "icin", "icinde", "iki", "ilgili", "ilk", "iptal",
    "isaretler", "islem", "istek", "istekleri", "kadar", "kalir", "karakter", "kayit",
    "kaydi", "kendi", "kisi", "kosul", "kosulu", "kullanilir", "kullanir", "kurulum",
    "maksimum", "mevcut", "neden", "olabilir", "olan", "olarak", "olmasi", "oldugu",
    "olusan", "olustu", "olusur", "onceki", "onemli", "sadece", "saglar", "satir",
    "sayisi", "sema", "semasi", "sinir", "siniri", "sirasinda", "soyler", "sonra",
    "sonuc", "sunar", "surec", "suresi", "tarafindan", "tekrar", "tutar", "tum", "uzere",
    "uzerinde", "vardir", "veya", "verir", "veritabani", "yalnizca", "yapan", "yapar",
    "yapilir", "yani", "yeni", "yerine", "yoksa", "zaman", "zorunlu", "kural", "kurallar", "gerekli", "gereken",
}

# Technical vocabulary that no English dictionary carries. Only consulted by --strict.
TECHNICAL = set("""api http json sql url uuid jwt rls ttl cpu io db postgres redis lua fiber
goroutine goroutines middleware tenant multi runtime config env repo struct async sync
idempotency backoff jitter singleflight namespace timestamp schema metadata bool func var
const param auth cors todo cache lifecycle enum crud dto docker compose yaml prometheus otel
latency scan unmarshal observability frontend backend fallback workspace timeout timeouts
retries setup mock fixture zerolog localhost hostname ip dns rollback website webhook email
login logout cancelled cancelling formatted properties planning paid scanned submitted rdb
ctx debug byte validator avatar categories currencies entries info payload callback callbacks
endpoint endpoints blog catalog superuser superusers behaviour repositories dependencies
queries verified verifies database policies marketplace iyzico bcrypt lowercase overridden
capabilities favicon glob openapi usecase whitespace timezone timeline unpublish unpublishes
unpublishing unscoped tenantctx ratelimit refactor resetting parameterised prepended satisfies
trimmed identifies materialise strongest availabilities benchmarks blew blipped cancellable
controlled customized debugf directories errorf execs fatalf infof warnf demo cmd hitting
planned programming escalation doesn earlier lowest multiplied nullability retried applies
became chargeback cta asc desc tokenized uppercase mastercard screenshot keywords homepage
deutsch anasayfa baseline godoc oapi codegen onboarding filename normalised sqlmock plaintext
ffffff int benchmark cardinality etc filesystem optimisation soonest subdomain
supplies uninstall""".split())

SUFFIXES = ("s", "es", "ed", "d", "ing", "er", "ers", "ly", "tion", "ation", "al", "able", "ible")
DICTIONARIES = ("/usr/share/dict/words", "/usr/dict/words", "/usr/share/dict/american-english")

# Sample i18n content in migration 026 is Turkish on purpose: it is the data a
# multi-language CMS stores, not prose about the code.
SKIP_LINES = {
    ("migrations/026_create_cms_tables.up.sql", 103),
    ("migrations/026_create_cms_tables.up.sql", 113),
    ("migrations/026_create_cms_tables.up.sql", 124),
}

# Comment markers, plus the strings a shell script or Makefile prints. In Go and SQL the
# prose lives in comments; in a shell script it lives in echo, and a Turkish message
# printed at the terminal is just as visible to a reader as one in the source.
COMMENT = re.compile(r"(?://|--|#)\s?(.*)$")
PRINTED = re.compile(r"""["']([^"']{8,})["']""")
BACKTICKED = re.compile(r"`[^`]*`")

# Sentence punctuation is stripped before the code filter runs. Without this step a word
# ending a sentence ("zorunludur.") looks like a qualified identifier and is discarded,
# which silently exempted the last word of most comments.
SENTENCE_PUNCT = re.compile(r"(?<=[A-Za-z])[.,;:!?]+(?=\s|$)")

# Anything still carrying these characters is an identifier, a path or an expression.
CODEISH = re.compile(r"\S*[/._(){}\[\]<>=$%:@|\\]\S*")


def is_turkish(token):
    """Match a marker, or an inflected form of one.

    Turkish is agglutinative: "zorunlu" appears as "zorunludur", "zorunlulugu" and so
    on. Exact matching against the marker list let "zorunludur" through, so a marker of
    four characters or more also matches as a prefix.
    """
    token = token.lower()
    if token in TURKISH_MARKERS:
        return True
    return any(len(m) >= 4 and token.startswith(m) for m in TURKISH_MARKERS)


def comment_tokens():
    """Yield (path, line number, raw line, tokens) for every comment in the tree."""
    files = [
        f for f in subprocess.check_output(["git", "ls-files"], text=True).split()
        if f.endswith((".go", ".sql", ".yml", ".yaml", ".sh", ".py")) or f == "Makefile"
    ]
    for path in files:
        try:
            lines = open(path, encoding="utf-8").read().split("\n")
        except (OSError, UnicodeDecodeError):
            continue
        for number, line in enumerate(lines, 1):
            if (path, number) in SKIP_LINES:
                continue
            parts = []
            if match := COMMENT.search(line):
                parts.append(match.group(1))
            if path.endswith((".sh",)) or path == "Makefile":
                parts.extend(PRINTED.findall(line))
            if not parts:
                continue
            text = BACKTICKED.sub(" ", " ".join(parts))
            text = CODEISH.sub(" ", SENTENCE_PUNCT.sub(" ", text))
            tokens = [
                tok for tok in re.findall(r"[A-Za-z]{3,}", text)
                if not any(c.isupper() for c in tok[1:])
            ]
            if tokens:
                yield path, number, line.strip()[:100], tokens


def load_dictionary():
    for candidate in DICTIONARIES:
        if os.path.exists(candidate):
            with open(candidate, encoding="latin-1") as fh:
                return {line.strip().lower() for line in fh}, candidate
    return None, None


def in_dictionary(token, words):
    if token in words or token in TECHNICAL or len(token) <= 2:
        return True
    for suffix in SUFFIXES:
        if token.endswith(suffix):
            stem = token[: -len(suffix)]
            if any(c in words or c in TECHNICAL for c in (stem, stem + "e", stem + "y")):
                return True
    return False


def report(title, findings):
    for path, rows in sorted(findings.items()):
        for number, text, tokens in rows:
            print(f"{path}:{number}: {text}\n    {title}: {', '.join(sorted(set(tokens)))}")


def main():
    strict = "--strict" in sys.argv

    markers = collections.defaultdict(list)
    dictionary_misses = collections.defaultdict(list)

    words, source = load_dictionary() if strict else (None, None)

    for path, number, text, tokens in comment_tokens():
        hits = [t for t in tokens if is_turkish(t)]
        if hits:
            markers[path].append((number, text, hits))
        if words is not None:
            misses = [t for t in tokens if not in_dictionary(t.lower(), words)]
            if misses:
                dictionary_misses[path].append((number, text, misses))

    if markers:
        report("turkish", markers)
        print(f"\nlanguage check: {sum(len(r) for r in markers.values())} non-English comment(s)")
        return 1

    if strict:
        if words is None:
            print("language check: no dictionary installed, ran the marker pass only")
        elif dictionary_misses:
            report("unrecognised", dictionary_misses)
            print(f"\nlanguage check: {sum(len(r) for r in dictionary_misses.values())} "
                  f"token(s) in no dictionary ({source}). Add real terms to TECHNICAL.")
            return 1

    print("language check: clean")
    return 0


if __name__ == "__main__":
    sys.exit(main())
