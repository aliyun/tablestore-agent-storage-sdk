use std::fmt;
use std::sync::{Arc, Mutex};

/// Long-lived credentials used to sign every request.
///
/// `Debug` is implemented manually so [`Credentials::access_key_secret`]
/// and [`Credentials::security_token`] are redacted in logs.
#[derive(Clone)]
pub struct Credentials {
    /// Aliyun AccessKey ID.
    pub access_key_id: String,
    /// Aliyun AccessKey Secret.
    pub access_key_secret: String,
    /// Optional STS session token; sent as `x-ots-ststoken` when set.
    pub security_token: Option<String>,
}

impl fmt::Debug for Credentials {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("Credentials")
            .field("access_key_id", &self.access_key_id)
            .field("access_key_secret", &"[redacted]")
            .field(
                "security_token",
                &self.security_token.as_ref().map(|_| "[redacted]"),
            )
            .finish()
    }
}

impl Credentials {
    /// Convenience constructor for the common two-field case.
    pub fn new(access_key_id: impl Into<String>, access_key_secret: impl Into<String>) -> Self {
        Self {
            access_key_id: access_key_id.into(),
            access_key_secret: access_key_secret.into(),
            security_token: None,
        }
    }
}

/// Per-request mutable signing context.
///
/// `sign_date` is set by V4 after the signing key is derived, and read back
/// when verifying the response signature on the same request. V2 ignores it.
#[derive(Debug, Clone)]
pub struct RequestContext {
    credentials: Credentials,
    sign_date: Arc<Mutex<Option<String>>>,
}

impl RequestContext {
    /// Build a context from credentials with no preset sign date.
    pub fn new(credentials: Credentials) -> Self {
        Self {
            credentials,
            sign_date: Arc::new(Mutex::new(None)),
        }
    }

    /// Build a context with a fixed sign date — used for V4 test fixtures.
    pub fn with_sign_date(credentials: Credentials, sign_date: impl Into<String>) -> Self {
        Self {
            credentials,
            sign_date: Arc::new(Mutex::new(Some(sign_date.into()))),
        }
    }

    /// Borrow the underlying credentials.
    pub fn credentials(&self) -> &Credentials {
        &self.credentials
    }

    /// Read the sign date captured by V4 signing, if any.
    pub fn sign_date(&self) -> Option<String> {
        self.sign_date
            .lock()
            .expect("sign_date mutex poisoned")
            .clone()
    }

    /// Record the sign date used while constructing the request signature.
    pub fn set_sign_date(&self, sign_date: impl Into<String>) {
        *self.sign_date.lock().expect("sign_date mutex poisoned") = Some(sign_date.into());
    }
}
