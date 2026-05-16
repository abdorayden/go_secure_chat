"""
generate_report.py
Converts REPORT.md into a professionally typeset PDF that looks like
internal engineering documentation (think Stripe/Linear internal docs).

Dependencies:
    pip install markdown weasyprint pygments

Usage:
    python generate_report.py
Output:
    server_report.pdf
"""

import datetime
from markdown import markdown
from weasyprint import HTML, CSS


# ── 1. Read source ────────────────────────────────────────────────────────────

with open("../REPORT.md", "r", encoding="utf-8") as f:
    md_text = f.read()


# ── 2. Prepend metadata block ─────────────────────────────────────────────────

today = datetime.date.today().strftime("%B %d, %Y")

metadata_block = f"""\
# Server Implementation Report

<div class="doc-meta">
  <span><strong>Date:</strong> {today}</span>
  <span><strong>Author:</strong> Abdenour Souane</span>
  <span><strong>Classification:</strong> Internal</span>
</div>

---

"""

md_full = metadata_block + md_text


# ── 3. Markdown → HTML ────────────────────────────────────────────────────────

body_html = markdown(
    md_full,
    extensions=[
        "fenced_code",
        "tables",
        "codehilite",
        "toc",
        "nl2br",
        "sane_lists",
    ],
    extension_configs={
        "codehilite": {
            "css_class": "highlight",
            "guess_lang": False,
            "noclasses": True,  # inline styles — no external pygments CSS needed
        },
        "toc": {
            "title": "Contents",
            "toc_depth": 3,
        },
    },
)


# ── 4. Full HTML document ─────────────────────────────────────────────────────

# Accent palette (single source of truth — change here to retheme)
ACCENT      = "#2c3e50"   # navy — borders, table headers, links, h3
ACCENT_LITE = "#3d5166"   # slightly lighter navy for hover/active states
BODY_TEXT   = "#2d2d2d"   # near-black body copy
HEAD_TEXT   = "#1a1a1a"   # headings
MUTED       = "#666666"   # meta, captions
RULE_COLOR  = "#e0e0e0"   # horizontal rules, table borders
ROW_ALT     = "#fafafa"   # alternating table row
CODE_BG     = "#f8f8f8"   # code block background
CODE_BORDER = ACCENT
CODE_INLINE_FG = "#c7254e"  # reddish inline code text
CODE_INLINE_BG = "#fdf0f2"  # very light rose inline code background

html_page = f"""<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<style>

/* ═══════════════════════════════════════════════════════════════
   PAGE SETUP
═══════════════════════════════════════════════════════════════ */

@page {{
    size: A4;
    margin: 2.5cm 2cm 2.5cm 2cm;

    /* Running header — suppressed on first page via :first rule */
    @top-center {{
        content: "Server Implementation Report";
        font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;
        font-size: 8.5pt;
        color: {MUTED};
        font-weight: 400;
        letter-spacing: 0.02em;
    }}

    /* Page footer */
    @bottom-right {{
        content: "Page " counter(page) " of " counter(pages);
        font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;
        font-size: 8pt;
        color: #999;
    }}

    @bottom-left {{
        content: "Confidential — Internal Use Only";
        font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;
        font-size: 8pt;
        color: #bbb;
        font-style: italic;
    }}
}}

/* No running header on the title page */
@page :first {{
    @top-center {{ content: none; }}
    @bottom-left {{ content: none; }}
    margin-top: 3cm;
}}


/* ═══════════════════════════════════════════════════════════════
   RESET & BASE
═══════════════════════════════════════════════════════════════ */

*, *::before, *::after {{
    box-sizing: border-box;
}}

html {{
    font-size: 10.5pt;
    -webkit-text-size-adjust: 100%;
}}

body {{
    font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;
    font-size: 1rem;          /* inherits 10.5pt */
    line-height: 1.75;
    color: {BODY_TEXT};
    background: #ffffff;
    margin: 0;
    padding: 0;
    text-rendering: optimizeLegibility;
    font-feature-settings: "kern" 1, "liga" 1;
    hyphens: auto;
    hyphenate-limit-chars: 6 3 3;
    hyphenate-limit-lines: 2;
    widows: 3;
    orphans: 3;
}}


/* ═══════════════════════════════════════════════════════════════
   DOCUMENT TITLE (h1 — appears only once, acts as cover header)
═══════════════════════════════════════════════════════════════ */

h1 {{
    font-size: 26pt;
    font-weight: 700;
    color: {HEAD_TEXT};
    line-height: 1.15;
    letter-spacing: -0.4pt;
    margin: 0 0 18px 0;
    padding-bottom: 16px;
    border-bottom: 3px solid {ACCENT};
    page-break-after: avoid;
}}


/* ═══════════════════════════════════════════════════════════════
   METADATA LINE (injected as .doc-meta div)
═══════════════════════════════════════════════════════════════ */

.doc-meta {{
    display: flex;
    flex-wrap: wrap;
    gap: 0 28px;
    margin: 10px 0 28px 0;
    font-size: 9pt;
    color: {MUTED};
    line-height: 1.5;
}}

.doc-meta strong {{
    color: #444;
    font-weight: 600;
}}


/* ═══════════════════════════════════════════════════════════════
   TABLE OF CONTENTS (generated by the toc extension)
═══════════════════════════════════════════════════════════════ */

.toc {{
    background: #f5f7f9;
    border: 1px solid {RULE_COLOR};
    border-left: 4px solid {ACCENT};
    border-radius: 3px;
    padding: 18px 22px 18px 26px;
    margin: 30px 0 38px 0;
    page-break-inside: avoid;
    font-size: 9.5pt;
    line-height: 1.8;
}}

.toc > ul::before {{
    content: "Table of Contents";
    display: block;
    font-size: 10.5pt;
    font-weight: 700;
    color: {ACCENT};
    letter-spacing: 0.02em;
    text-transform: uppercase;
    margin-bottom: 10px;
    padding-bottom: 8px;
    border-bottom: 1px solid {RULE_COLOR};
}}

.toc ul {{
    margin: 0;
    padding-left: 16px;
    list-style: none;
}}

.toc > ul {{
    padding-left: 0;
}}

.toc li {{
    margin: 2px 0;
}}

.toc a {{
    color: {BODY_TEXT};
    text-decoration: none;
    border-bottom: none;
}}

.toc a:hover {{
    color: {ACCENT};
}}

/* Indent nested levels */
.toc ul ul {{
    padding-left: 16px;
    font-size: 9pt;
    color: {MUTED};
    margin-top: 2px;
}}

.toc ul ul ul {{
    padding-left: 14px;
    font-size: 8.5pt;
}}


/* ═══════════════════════════════════════════════════════════════
   SECTION HEADINGS
═══════════════════════════════════════════════════════════════ */

h2 {{
    font-size: 15pt;
    font-weight: 700;
    color: {HEAD_TEXT};
    letter-spacing: -0.2pt;
    margin: 44px 0 14px 0;
    padding-bottom: 8px;
    border-bottom: 1.5px solid {RULE_COLOR};
    line-height: 1.25;
    page-break-after: avoid;
}}

h3 {{
    font-size: 12pt;
    font-weight: 600;
    color: {ACCENT};
    margin: 32px 0 10px 0;
    line-height: 1.3;
    page-break-after: avoid;
}}

h4 {{
    font-size: 10.5pt;
    font-weight: 600;
    color: #3a3a3a;
    margin: 24px 0 8px 0;
    line-height: 1.35;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    page-break-after: avoid;
}}

h5, h6 {{
    font-size: 10pt;
    font-weight: 600;
    color: {MUTED};
    margin: 18px 0 6px 0;
    page-break-after: avoid;
}}


/* ═══════════════════════════════════════════════════════════════
   PARAGRAPH & INLINE TEXT
═══════════════════════════════════════════════════════════════ */

p {{
    margin: 0 0 14px 0;
    text-align: justify;
}}

strong, b {{
    font-weight: 600;
    color: {HEAD_TEXT};
}}

em, i {{
    font-style: italic;
}}

a {{
    color: {ACCENT};
    text-decoration: none;
    border-bottom: 1px solid rgba(44, 62, 80, 0.35);
}}

hr {{
    border: none;
    border-top: 1px solid {RULE_COLOR};
    margin: 36px 0;
}}


/* ═══════════════════════════════════════════════════════════════
   LISTS
═══════════════════════════════════════════════════════════════ */

ul, ol {{
    margin: 8px 0 16px 0;
    padding-left: 24px;
}}

li {{
    margin-bottom: 5px;
    line-height: 1.7;
    text-align: left;            /* don't justify list items */
}}

li > ul,
li > ol {{
    margin-top: 4px;
    margin-bottom: 4px;
}}

ul {{
    list-style-type: disc;
}}

ul ul {{
    list-style-type: circle;
}}

ul ul ul {{
    list-style-type: square;
}}

/* Prevent list breaks */
ul, ol {{
    page-break-inside: avoid;
}}


/* ═══════════════════════════════════════════════════════════════
   INLINE CODE
═══════════════════════════════════════════════════════════════ */

code {{
    font-family: "SF Mono", Menlo, Monaco, Consolas, "Liberation Mono",
                 "Courier New", monospace;
    font-size: 8.5pt;
    line-height: 1;
    color: {CODE_INLINE_FG};
    background: {CODE_INLINE_BG};
    border: 1px solid rgba(199, 37, 78, 0.15);
    border-radius: 3px;
    padding: 1.5pt 4pt;
    white-space: nowrap;
    /* Don't inherit hyphens */
    hyphens: none;
}}


/* ═══════════════════════════════════════════════════════════════
   CODE BLOCKS  (weasyprint renders <pre><code> via codehilite
                 with inline Pygments styles)
═══════════════════════════════════════════════════════════════ */

pre {{
    font-family: "SF Mono", Menlo, Monaco, Consolas, "Liberation Mono",
                 "Courier New", monospace;
    font-size: 8.5pt;
    line-height: 1.6;
    background: {CODE_BG};
    border: 1px solid #e2e2e2;
    border-left: 4px solid {CODE_BORDER};
    border-radius: 0 3px 3px 0;
    padding: 14px 16px;
    margin: 16px 0;
    overflow-x: auto;
    white-space: pre;
    word-wrap: normal;
    page-break-inside: avoid;
    color: #333;
    hyphens: none;
}}

/* Reset inline-code styles inside a block */
pre code {{
    font-family: inherit;
    font-size: inherit;
    color: inherit;
    background: transparent;
    border: none;
    border-radius: 0;
    padding: 0;
    white-space: pre;
}}

/* Pygments highlight wrapper */
.highlight {{
    margin: 0;
    background: transparent;
}}

/* Language label — uses a data attribute set by the extension */
pre[data-lang]::before {{
    content: attr(data-lang);
    display: block;
    font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;
    font-size: 7.5pt;
    font-weight: 600;
    color: #999;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    margin-bottom: 10px;
    padding-bottom: 6px;
    border-bottom: 1px solid #e2e2e2;
}}


/* ═══════════════════════════════════════════════════════════════
   TABLES
═══════════════════════════════════════════════════════════════ */

table {{
    width: 100%;
    border-collapse: collapse;
    margin: 20px 0 28px 0;
    font-size: 9.5pt;
    page-break-inside: avoid;
    text-align: left;
}}

thead {{
    background-color: {ACCENT};
    color: #ffffff;
}}

thead tr {{
    page-break-after: avoid;
}}

th {{
    padding: 10px 13px;
    font-weight: 600;
    font-size: 9pt;
    letter-spacing: 0.02em;
    border: none;
    text-align: left;
    hyphens: none;
    white-space: nowrap;
}}

td {{
    padding: 8px 13px;
    border-bottom: 1px solid {RULE_COLOR};
    vertical-align: top;
    color: {BODY_TEXT};
}}

/* Remove bottom border from last row so table doesn't look double-ruled */
tbody tr:last-child td {{
    border-bottom: none;
}}

/* Alternating row backgrounds */
tbody tr:nth-child(even) td {{
    background: {ROW_ALT};
}}

/* No grid — only bottom separator */
th, td {{
    border-left: none;
    border-right: none;
    border-top: none;
}}

thead th:first-child {{ border-radius: 3px 0 0 0; }}
thead th:last-child  {{ border-radius: 0 3px 0 0; }}


/* ═══════════════════════════════════════════════════════════════
   BLOCKQUOTES
═══════════════════════════════════════════════════════════════ */

blockquote {{
    margin: 20px 0;
    padding: 12px 18px;
    border-left: 4px solid {ACCENT};
    background: #f5f7f9;
    color: #444;
    font-style: italic;
    border-radius: 0 3px 3px 0;
    page-break-inside: avoid;
}}

blockquote p {{
    margin: 0;
    text-align: left;
}}

blockquote p + p {{
    margin-top: 8px;
}}

blockquote strong {{
    font-style: normal;
}}


/* ═══════════════════════════════════════════════════════════════
   DEFINITION LISTS  (sometimes used in Go docs)
═══════════════════════════════════════════════════════════════ */

dl {{
    margin: 14px 0;
}}

dt {{
    font-weight: 600;
    color: {ACCENT};
    margin-top: 10px;
}}

dd {{
    margin: 3px 0 6px 22px;
    color: {BODY_TEXT};
}}


/* ═══════════════════════════════════════════════════════════════
   MISCELLANEOUS
═══════════════════════════════════════════════════════════════ */

/* Syntax-highlighted spans (inline Pygments styles override these;
   these are only fallback defaults) */
.highlight .k  {{ color: #0000dd; font-weight: bold; }}
.highlight .s  {{ color: #008800; }}
.highlight .c  {{ color: #888888; font-style: italic; }}
.highlight .n  {{ color: {BODY_TEXT}; }}

/* Utility: prevent widows on the last section */
.no-widow {{
    widows: 4;
    orphans: 4;
}}

/* ── END-OF-DOCUMENT FOOTER NOTE ── */
.doc-footer {{
    margin-top: 56px;
    padding-top: 14px;
    border-top: 1px solid {RULE_COLOR};
    font-size: 8pt;
    color: #aaa;
    font-style: italic;
    line-height: 1.5;
    text-align: left;
}}

</style>
</head>
<body>

{body_html}

<div class="doc-footer">
    This document is intended for internal use by the engineering team.
    Do not distribute outside the organisation without prior written approval.
    For questions or corrections, contact the backend engineering team.
</div>

</body>
</html>"""


# ── 5. Render PDF ─────────────────────────────────────────────────────────────

output_path = "server_report.pdf"

HTML(string=html_page, base_url=".").write_pdf(
    output_path,
    presentational_hints=True,
)

print(f"✓  Generated: {output_path}")
