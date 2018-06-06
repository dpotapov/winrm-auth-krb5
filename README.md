# winrm-auth-krb5

Krb5 Transporter for the [masterzen's Go WinRM](https://github.com/masterzen/winrm) client.

Windows & Linux implementations are very different!
Please check https://github.com/dpotapov/go-spnego for details.

Installation:

```
go get github.com/dpotapov/winrm-auth-krb5
```

Usage:

```
endpoint := winrm.NewEndpoint(*host, *port, false, false, nil, nil, nil, 0)

winrm.DefaultParameters.TransportDecorator = func() winrm.Transporter {
    return &winrmkrb5.Transport{}
}

// Note, username/password pair in the NewClientWithParameters call is ignored
client, err := winrm.NewClientWithParameters(endpoint, "", "", winrm.DefaultParameters)
if err != nil {
    panic(err)
}

_, err = client.Run(flag.Arg(0), os.Stdout, os.Stderr)
if err != nil {
    panic(err)
}
```

Please check the full example in the `example` directory.