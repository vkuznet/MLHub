# MLHub documentation
MLHub is a machine learning service for open science. It is a platform for storing and publishing trained ML models coupled with an inference engine that delivers insights on demand. MLHub democratizes access to machine learning resources for communities from all scientific domains.

Using MLHub, researchers can easily:
* Upload, organize, and manage privacy settings on their own trained models.
* Publish thier models and assign DOIs with the click of a button.
* Search for models published by other researchers and generate citations.
* Run the inference engine to compute output predictions on any input dataset, using any model in the repository to which the researcher has access.

MLHub is more than just a reference library of published science. It can be directly used in machine learning workflows as tool for research itself. So, by incorporating a public service like MLHub early in their research process, scientists simplify the eventual task of making their published research FAIR-compliant.

## Architecture
MLHub supports all common MLaaS backend frameworks, including TensorFlow, PyTorch, Keras, and scikit-learn. It consists of the following components:
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
an uniform way to query these services.

## MLHub APIs
MLHub provides the following set of APIs:
- `/model/<name>` end-point provides the following methods:
  - `GET` HTTP request will retrieve ML meta-data for provide ML name, e.g.
```
# fetch meta-data info about ML model
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
```
  - `PUT` HTTP request will update exsiting ML entry in MLHub for provided
  ML meta-data JSON record
```
# post ML meta-data
curl -X PUT \
     -H "content-type: application/json" \
     -d '{"model": "mnist", "type": "TensorFlow", "meta": {"param": 1}}' \
     http://localhost:port/model/mnist
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

### ML model APIs
- `/upload` uploads ML model bundle
```
# upload ML model with its meta-data
curl -H 'Accept: application/json' \
    -F 'file=@/path/mnist.tar.gz' \
    -F 'model=mnist' \
    -F 'version=v1.1.1' \
    -F 'type=TensorFlow' \
    -F 'description=bla' \
    -F 'reference=http://site.com' \
    http://localhost:port/upload
```

- `/model/<model_name>/upload` uploads ML model bundle
```
# upload ML model for existing meta-data
curl -X POST -H "Content-Encoding: gzip" \
     -H "content-type: application/octet-stream" \
     --data-binary @./mnist.tar.gz \
     http://localhost:port/model/mnist/upload

```
- `/model/<model_name>/download` downloads ML model bundle
```
curl http://localhost:port/model/mnist/download
```
- `/model/<model_name>/predict` to get prediction from a given ML model.
```
# provide prediction for given input vector
curl -X GET \
     -H "content-type: application/json" \
     -d '{"input": [input values]}' \
     http://localhost:port/model/mnist/predict

# provide prediction for given image file
curl http://localhost:8083/model/mnist/predict \
     -F 'image=@./img4.png'
```

