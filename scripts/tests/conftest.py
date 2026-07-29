"""pytest 設定。scripts/tests/ から親の scripts/ 配下モジュールを import できるようにする."""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))
