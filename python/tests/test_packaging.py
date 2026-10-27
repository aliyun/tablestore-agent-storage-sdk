import os
import subprocess
import sys
from pathlib import Path

try:
    import tomllib
except ModuleNotFoundError:
    import tomli as tomllib

PYTHON_ROOT = Path(__file__).resolve().parents[1]
SOURCE_ROOT = PYTHON_ROOT / "src" / "tablestore_agent_storage"


def test_project_has_no_tablestore_dependency_and_declares_urllib3():
    project = tomllib.loads((PYTHON_ROOT / "pyproject.toml").read_text())
    dependencies = project["project"]["dependencies"]

    assert not any(item.lower().startswith("tablestore") for item in dependencies)
    assert any(item.lower().startswith("urllib3") for item in dependencies)


def test_source_does_not_import_tablestore_package():
    source = "\n".join(path.read_text() for path in SOURCE_ROOT.glob("*.py"))

    assert "import tablestore\n" not in source
    assert "from tablestore import" not in source


def test_real_package_exports_internal_errors_and_retries():
    code = """
import sys
import types
sys.modules['oss2'] = types.ModuleType('oss2')
import tablestore_agent_storage as sdk
print(sdk.__version__)
print(sdk.OTSClientError.__module__)
print(sdk.OTSServiceError.__module__)
print(sdk.DefaultRetryPolicy.__module__)
"""
    environment = dict(os.environ)
    environment["PYTHONPATH"] = str(PYTHON_ROOT / "src")

    result = subprocess.run(
        [sys.executable, "-c", code],
        check=True,
        capture_output=True,
        text=True,
        env=environment,
    )

    project = tomllib.loads((PYTHON_ROOT / "pyproject.toml").read_text())
    assert result.stdout.splitlines() == [
        project["project"]["version"],
        "tablestore_agent_storage.errors",
        "tablestore_agent_storage.errors",
        "tablestore_agent_storage.retry",
    ]
