module kb-product-matrix-runner

go 1.25.7

require github.com/matrixorigin/matrixflow/sdk/go-sdk v0.0.0

require (
	github.com/matrixorigin/matrixflow/sdk/product-model v0.0.0 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

replace github.com/matrixorigin/matrixflow/sdk/go-sdk => ../../../sdk/go-sdk

replace github.com/matrixorigin/matrixflow/sdk/product-model => ../../../sdk/product-model
