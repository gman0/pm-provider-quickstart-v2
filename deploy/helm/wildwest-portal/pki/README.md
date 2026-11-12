Static PKI material that we use for the wildwest portal. This is also what's inside the ContentConfiguration.

```
   mkdir -p deploy/helm/wildwest-portal/pki && cd deploy/helm/wildwest-portal/pki && \
   openssl req -x509 -newkey rsa:2048 -nodes -days 3650 \
     -keyout ca.key -out ca.crt \
     -subj "/CN=wildwest-portal-dev-ca" \
     -addext "basicConstraints=critical,CA:TRUE" \
     -addext "keyUsage=critical,keyCertSign,cRLSign" 2>&1 | tail -3 && \
   openssl req -newkey rsa:2048 -nodes \
     -keyout tls.key -out tls.csr \
     -subj "/CN=wildwest.127-0-0-1.sslip.io" 2>&1 | tail -3 && \
   cat > san.ext <<'EOF'
   subjectAltName = DNS:wildwest.127-0-0-1.sslip.io
   extendedKeyUsage = serverAuth
   EOF
   openssl x509 -req -in tls.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
     -out tls.crt -days 3650 -extfile san.ext 2>&1 | tail -3 && \
   rm -f tls.csr ca.key ca.srl san.ext && ls
```
