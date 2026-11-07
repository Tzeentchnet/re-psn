"""Keep server theme tokens in sync with the project palette."""
import json
from pathlib import Path
import re
import unittest

ROOT = Path(__file__).resolve().parent.parent


class StatusPaletteTest(unittest.TestCase):
    def test_embedded_palette_matches_the_project_source(self):
        palette = json.loads((ROOT / 'docs/art/palette.json').read_text())['modes']
        css = (ROOT / 'server/status.css').read_text()
        blocks = re.findall(r'([^{}]+)\{([^{}]*--bg0:[^{}]+)\}', css)
        self.assertEqual(len(blocks), 3)
        for selector, body in blocks:
            mode = 'light' if 'light' in selector or 'not(' in selector else 'dark'
            actual = dict(re.findall(r'--([a-z0-9]+):\s*(#[0-9a-f]{6})', body))
            self.assertEqual(actual, palette[mode])
