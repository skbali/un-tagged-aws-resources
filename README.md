# un-tagged-aws-resources

## go compile
```bash
GOARCH=arm64 GOOS=linux go build -tags lambda.norpc -o build/bootstrap main.go
OR if you want an x86_64 lambda function
GOARCH=amd64 GOOS=linux go build -tags lambda.norpc -o build/bootstrap main.go
```

## terraform
```bash
 terraform init
 terraform plan
 terraform apply -auto-approve
``` 