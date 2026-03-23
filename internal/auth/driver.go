package auth

// Credentials holds basic credential representation
type Credentials struct {
    Username string
    Password string
}

// AuthDriver interface abstracts authentication mechanism for Reg.ru
type AuthDriver interface {
    // PrepareAuth returns headers or form values to include in a request
    PrepareAuth() (map[string]string, error)
}

// TokenDriver is a simple username/password driver
type TokenDriver struct {
    Creds Credentials
}

func (t *TokenDriver) PrepareAuth() (map[string]string, error) {
    return map[string]string{
        "username": t.Creds.Username,
        "password": t.Creds.Password,
    }, nil
}

// RSASignatureDriver placeholder for RSA sig auth
type RSASignatureDriver struct {
    // private key or path to key
}

func (r *RSASignatureDriver) PrepareAuth() (map[string]string, error) {
    // TODO: implement signature generation
    return map[string]string{}, nil
}

