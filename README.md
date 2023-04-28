# MLHub
MLHub represents a hub for different MLaaS backends. It provides the following
functionality:
- MetaData service for pre-trained ML models
- A reverse proxy to different MLaaS backends:
```
                   | -> TFaaS
client --> MLHub --| -> PyTorch
             |     | -> Keras+ScikitLearn
             |
             |--------> MetaData service
```
Each ML backend server may have different set of APIs and MLHub provides
an uniform way to query these services. So far we support the following APIs:
- `/model/<name>` end-point provides the following methods:
  - `GET` HTTP request will retrieve ML meta-data for provide ML name, e.g.
```
curl http://localhost:port/model/mnist
```
  - `POST` HTTP request will create new ML entry in MLHub for provided
  ML meta-data JSON record and ML tarball
```
# post ML meta-data
curl -X POST \
     -H "content-type: application/json" \
     -d '{"model": "mnist", "type": "TensorFlow", "meta": {}}' \
     http://localhost:port/model/mnist

# upload ML model
curl -X POST -H "Content-Encoding: gzip" \
     -H "content-type: application/octet-stream" \
     --data-binary @./mnist.tar.gz \
     http://localhost:port/model/mnist/upload
```
  - `DELETE` HTTP request will delete ML entry in MLHub for provided ML name
```
curl -X DELETE \
     http://localhost:port/model/mnist
```
- `/models` to list existing ML models, GET HTTP request
```
# to get all ML models
curl http://localhost:port/models
```
- `/model/<model_name>/predict` to get prediction from a given ML model.
```
curl -X GET \
     -H "content-type: application/json" \
     -d '{"input": [input values]}' \
     http://localhost:port/model/mnist/predict
```
or
```
curl http://localhost:8083/model/mnist \
     -F 'image=@./img4.png'
```
