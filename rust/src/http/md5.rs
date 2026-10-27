use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
use md5::{Digest, Md5};

/// Base64-encoded MD5 used by `x-ots-contentmd5`.
pub fn md5_base64(content: &[u8]) -> String {
    let mut hasher = Md5::new();
    hasher.update(content);
    BASE64.encode(hasher.finalize())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn matches_known_fixture() {
        assert_eq!(md5_base64(b"hello"), "XUFAKrxLKna5cZ2REBfFkg==");
    }
}
