# login-svc-with-gcp
An example login service in Go using Google Cloud Platform for data storage
This service was written as a demo of how someone might go about implementing a login system as a Go microservice.
It contains the following functionality:

- Http handlers for health check and shutdown
- Allows adding a login via https POST (username/password combo) and storing it in a Google Firestore database
- Supports authenticating a username/password combo against that database and generating a JWT using a secret key stored in Google Secret Manager
- Contains an example of authorising a https request using the JWT as a bearer token

## Future Development
- Use the blueambertech/googlepubsub package to notify a message queue when a login is created