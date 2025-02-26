# Mutual TLS Authentication

Chain Core 1.2 introduces support for mutual TLS authentication. This means both Chain Core and your client application can authenticate each other using X.509 certificates via the TLS protocol. Client-side certificates may be used as an alternative to access tokens for authenticating client applications to Chain Core.

## Background

Mutual TLS authentication can be broken down into two parts: server authentication and client authentication.

**Server authentication** allows Chain Core to authenticate itself to client applications that connect to it. You can configure an instance of Chain Core with its own X.509 certificate and private key. To verify the server's certificate, the SDK must be configured with the certificate of the root CA that issued the server certificate.

**Client authentication** reverses these roles. The SDK is configured with its own certificate/key pair, and Chain Core is configured with the certificate of the root CA that issued the client certificate.

This guide will illustrate how to configure Chain Core and the Chain SDKs to use mutual TLS authentication. It assumes you have access to the following X.509 certificates and private keys:

- an X.509 certificate (and matching RSA private key) for the server
- the X.509 certificate of the root CA that issued the server's certificate
- an X.509 certificate (and matching RSA private key) for the client
- the X.509 certifcate of the root CA that issued the client's certificate

## Chain Core

#### Server authentication

Place the server's certificate and private key in the following locations:

[sidenote]

Note: `$CHAIN_CORE_HOME` can be set as an environment variable. It defaults to `$HOME/.chaincore`.

[/sidenote]

- certificate: `$CHAIN_CORE_HOME/tls.crt`
- private key: `$CHAIN_CORE_HOME/tls.key`

#### Client authentication

Set [`ROOT_CA_CERTS`](../reference/cored.md#extended-functionality) to the file path of the root CA certificate that issued the client's certificate.

Setting the root CA certificate allows Chain Core to validate and authenticate requests that use client certificates, but a client certificate will have no access to API resources by default. To provide access, you should create **authorization grants** in Chain Core that apply security policies to those certificates. See the [Authentication and Authorization guide](authentication-and-authorization.md#authorization) for more.

## Java SDK

The Java SDK's `Client` object exposes methods for mutual TLS configuration.

#### Server authentication

`Client#setTrustedCerts` configures the Client object with the certificate of the root CA that issued the server's certificate.

#### Client authentication

`Client#setX509KeyPair` configures the Client object with the client's certificate and private key.

### Example

```java
Client client = new Client.Builder()
  .setTrustedCerts(serverRootCACertPath)
  .setX509KeyPair(clientCertPath, clientCertPrivateKeyPath)
  .setURL("https://example.com")
  .build();
```

## Ruby SDK

The Ruby SDK's `Chain::Client` constructor accepts an `ssl_params` object for TLS configuration.

#### Server authentication

Set the `ca_file` attribute under `ssl_params` to the file path of the root CA that issued the server's certificate.

#### Client authentication

Set the `cert` (as an `OpenSSL::X509::Certificate` instance) and `key` (as an `OpenSSL::PKey::RSA` instance) attributes under `ssl_params`.

### Example

```ruby
cert = OpenSSL::X509::Certificate.new(client_cert_path)
key = OpenSSL::PKey::RSA.new(client_cert_private_key_path)

client = Chain::Client.new(
  url: "https://example.com",
  ssl_params: {
    cert: cert,
    key: key,
    ca_file: server_ca_cert_path
  }
)
```

## Node SDK

The Node SDK's `Client` constructor accepts an `agent` parameter (as an `https.Agent` instance) for TLS configuration.

#### Server authentication

Set the `ca` attribute to the certificate of the root CA that issued the server's certificate.

#### Client authentication

Set the `cert` and `key` attributes to the client's certificate and private key, respectively.

### Example

```js
const https = require('https')
const fs = require('fs')
const chain = require('chain-sdk')

const agent = new https.Agent({
  ca: fs.readFileSync(serverRootCACertPath),
  cert: fs.readFileSync(clientCertPath),
  key: fs.readFileSync(prclientCertPrivateKeyPath)
})

const client = new chain.Client({
  baseUrl: 'https://example.com',
  agent: agent
})
```
