# Test setup

### Client
```bash
# 1. At project root
make build

cd test
./certs.sh
mv *.pem ../build

cd ../build/
./proxy

# 2. Copy a go-valhaj v1.0.8-dev REPL binary to 'build/' and run it
```

### SAN
* Deployments on localhost / plain IP require a Subject Alternative Name (SAN) extension to the certificate.
* This means we'll have to add something along the lines of `IP:0.0.0.0` to the `*-ext.cnf` files.
* Keep in mind that we'll have to use this exact IP on both the client and the server side (e.g. `127.0.0.1` would not work).

