# Synthetic file fixtures

These files contain only synthetic SDK acceptance data. The document content is
the Cedar project and code 4827; the images are solid 8×8 RGB pixels. No customer
data or credentials are included.

PDF fixtures are marked binary in Git to preserve their byte offsets and
format-required whitespace.

`generate.py` creates PDF, DOCX, XLSX, PPTX, PNG, JPG and JPEG fixtures using
reportlab, python-docx, openpyxl, python-pptx and Pillow. It is an optional
maintenance tool, not a test/runtime dependency. Legacy Office files were
converted with LibreOffice:

```sh
cd go/e2e/testdata/formats
python3 generate.py
soffice --headless --convert-to doc --outdir . source.docx
soffice --headless --convert-to xls --outdir . source.xlsx
soffice --headless --convert-to ppt --outdir . source.pptx
```

The E2E suite validates full decoded byte equality, SHA-256 and size against the
checked-in files. It rejects invalid signatures and extension/media mismatches.
Wiki acceptance uses the PDF's unique code as a fact that must be published and
retrieved with PDF provenance; the Markdown input does not contain that code.
