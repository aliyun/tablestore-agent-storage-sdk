import sys
import types
from pathlib import Path

# Unit tests load focused source modules without executing the legacy package
# __init__, which imports the dependency this change removes. Packaging tests
# exercise the real package in a clean subprocess after integration.
PACKAGE_ROOT = Path(__file__).resolve().parents[1] / "src" / "tablestore_agent_storage"
package = types.ModuleType("tablestore_agent_storage")
package.__path__ = [str(PACKAGE_ROOT)]
sys.modules.setdefault("tablestore_agent_storage", package)
