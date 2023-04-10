# MLaaSProxy
A reverse proxy server for MLaaS backends:
```
                        | -> TFaaS
client --> MLaaSProxy --| -> PyTorch
                        | -> Keras+ScikitLearn
```
Each ML backend server may have different set of APIs and MLaaSProxy provides
an uniform way to query these services. So far we support the following APIs:
- `/upload` to upload ML tarball, POST HTTP request with ML model payload, e.g.
```
curl -X POST -H "Content-Encoding: gzip" \
     -H "content-type: application/octet-stream" \
     --data-binary @./mnist.tar.gz \
     http://localhost:port/upload
```
- `/models/<model_name>` to delete existing ML model, DELETE HTTP request
```
curl -X DELETE \
     http://localhost:port/models/mnist
```
- `/models/<model_name>` to list existing ML models, GET HTTP request
```
# to get all ML models
curl http://localhost:port/models

# to get concrete ML model
curl http://localhost:port/models/mnist
```
- `/models/<model_name>` to get prediction from a given ML model.
```
curl -X GET \
     -H "content-type: application/json" \
     -d '{"input": [input values]}' \
     http://localhost:port/models/mnist
```
or
```
curl http://localhost:8083/models/mnist \
     -F 'image=@./img4.png'
```
