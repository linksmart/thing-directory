# Known Issues of CAS 4.1.0-SNAPSHOT
* DELETE /cas/oauth2.0/profile, and /cas/p3/serviceValidate
    always respond with status 200 with a json error message for bad requests
* DELETE /cas/v1/tickets/TGT responds with status 200 if TGT expired or invalid
* GET /cas/p3/serviceValidate?service=$1&ticket=$2 does not return user attributes
* DELETE /cas/v1/tickets/TGT responds with status 500 on TGT deletion
* DELETE /cas/v1/tickets/TGT only gets triggered after calls to 
*   GET /cas/p3/serviceValidate?service=$1&ticket=$2
*   This will be fine as long as service token validation is done using CAS Protocol
# OAUTH:
* DELETE /cas/v1/tickets/TGT does not expire the ticket
* /cas/oauth2.0/profile does not return user attributes