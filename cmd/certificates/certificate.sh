openssl ecparam -genkey -name prime256v1 -out key$2.pem
openssl req -x509 -new -nodes -subj "/CN=$2" -addext "subjectAltName = IP.1:$1" -key key$2.pem -out cert$2.pem -sha256 -days 365
