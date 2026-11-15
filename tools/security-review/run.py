#!/usr/bin/env python3
"""Reproduce the 3.0 review findings without adding tests to production packages.

These checks pass when the reviewed vulnerability or bug is reproduced. They
are review evidence, not regression tests defining the desired behavior.
"""

import json
from pathlib import Path
import subprocess
import sys
import tempfile


def main() -> int:
    review_dir = Path(__file__).resolve().parent
    backend_dir = review_dir.parents[1] / "backend"
    replacements = {
        str(backend_dir / "handlers/security_review_repro_test.go"):
            str(review_dir / "repro/handlers_test.go"),
        str(backend_dir / "services/security_review_repro_test.go"):
            str(review_dir / "repro/services_test.go"),
    }
    print("Review reproductions: PASS means the reported issue still exists.", flush=True)
    with tempfile.TemporaryDirectory(prefix="tunnel-manager-v3-review-") as directory:
        overlay = Path(directory) / "overlay.json"
        overlay.write_text(json.dumps({"Replace": replacements}), encoding="utf-8")
        result = subprocess.run(
            ["go", "test", f"-overlay={overlay}", "./handlers", "./services",
             "-run", "^TestSecurityReview", "-v", "-count=1"],
            cwd=backend_dir,
            check=False,
        )
        return result.returncode


if __name__ == "__main__":
    sys.exit(main())
