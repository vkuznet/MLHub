# MLHub documentation
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
an uniform way to query these services.
