package wot

import (
	"time"
)

const MediaTypeThingDescription = "application/td+json"

/*
 This file has go models for Web Of Things (WoT) Things Description following : https://www.w3.org/TR/2019/CR-wot-thing-description-20191106/ (W3C Candidate Recommendation 6 November 2019)
*/

type any = interface{}

// ThingDescription is the structured data describing a Thing
type ThingDescription struct {

	// JSON-LD keyword to define short-hand names called terms that are used throughout a TD document.
	Context any `json:"@context"`

	// JSON-LD keyword to label the object with semantic tags (or types).
	Type any `json:"@type,omitempty"`

	// Identifier of the Thing in form of a URI [RFC3986] (e.g., stable URI, temporary and mutable URI, URI with local IP address, URN, etc.).
	ID AnyURI `json:"id,omitempty"`

	// Provides a human-readable title (e.g., display a text for UI representation) based on a default language.
	Title string `json:"title"`

	// Provides multi-language human-readable titles (e.g., display a text for UI representation in different languages).
	Titles map[string]string `json:"titles,omitempty"`

	// Provides additional (human-readable) information based on a default language
	Description string `json:"description,omitempty"`

	// Can be used to support (human-readable) information in different languages.
	Descriptions map[string]string `json:"descriptions,omitempty"`

	// Provides version information.
	Version *VersionInfo `json:"version,omitempty"`

	// Provides information when the TD instance was created.
	Created time.Time `json:"created,omitempty"`

	// Provides information when the TD instance was last modified.
	Modified time.Time `json:"modified,omitempty"`

	// Provides information about the TD maintainer as URI scheme (e.g., mailto [RFC6068], tel [RFC3966], https).
	Support AnyURI `json:"support,omitempty"`

	/*
		Define the base URI that is used for all relative URI references throughout a TD document. In TD instances, all relative URIs are resolved relative to the base URI using the algorithm defined in [RFC3986].

		base does not affect the URIs used in @context and the IRIs used within Linked Data [LINKED-DATA] graphs that are relevant when semantic processing is applied to TD instances.
	*/
	Base string `json:"base,omitempty"`

	// All Property-based Interaction Affordances of the Thing.
	Properties map[string]PropertyAffordance `json:"properties,omitempty"`

	// All Action-based Interaction Affordances of the Thing.
	Actions map[string]ActionAffordance `json:"actions,omitempty"`

	// All Event-based Interaction Affordances of the Thing.
	Events map[string]EventAffordance `json:"events,omitempty"`

	// Provides Web links to arbitrary resources that relate to the specified Thing Description.
	Links []Link `json:"links,omitempty"`

	// Set of form hypermedia controls that describe how an operation can be performed. Forms are serializations of Protocol Bindings. In this version of TD, all operations that can be described at the Thing level are concerning how to interact with the Thing's Properties collectively at once.
	Forms []Form `json:"forms,omitempty"`

	// Set of security definition names, chosen from those defined in securityDefinitions. These must all be satisfied for access to resources
	Security any `json:"security"`

	// Set of named security configurations (definitions only). Not actually applied unless names are used in a security name-value pair.
	SecurityDefinitions map[string]SecurityScheme `json:"securityDefinitions"`
}

/*Metadata of a Thing that shows the possible choices to Consumers, thereby suggesting how Consumers may interact with the Thing.
There are many types of potential affordances, but W3C WoT defines three types of Interaction Affordances: Properties, Actions, and Events.*/
type InteractionAffordance struct {
	// JSON-LD keyword to label the object with semantic tags (or types).
	Type any `json:"@type,omitempty"`

	// Provides a human-readable title (e.g., display a text for UI representation) based on a default language.
	Title string `json:"title,omitempty"`

	// Provides multi-language human-readable titles (e.g., display a text for UI representation in different languages).
	Titles map[string]string `json:"titles,omitempty"`

	// Provides additional (human-readable) information based on a default language
	Description string `json:"description,omitempty"`

	// Can be used to support (human-readable) information in different languages.
	Descriptions map[string]string `json:"descriptions,omitempty"`

	/*
		Set of form hypermedia controls that describe how an operation can be performed. Forms are serializations of Protocol Bindings.
		When a Form instance is within an ActionAffordance instance, the value assigned to op MUST be invokeaction.
		When a Form instance is within an EventAffordance instance, the value assigned to op MUST be either subscribeevent, unsubscribeevent, or both terms within an Array.
		When a Form instance is within a PropertyAffordance instance, the value assigned to op MUST be one of readproperty, writeproperty, observeproperty, unobserveproperty or an Array containing a combination of these terms.

	*/
	Forms []Form `json:"forms"`

	// Define URI template variables as collection based on DataSchema declarations.
	UriVariables map[string]DataSchema `json:"uriVariables,omitempty"`
}

/*
An Interaction Affordance that exposes state of the Thing. This state can then be retrieved (read) and optionally updated (write).
Things can also choose to make Properties observable by pushing the new state after a change.
*/
type PropertyAffordance struct {
	InteractionAffordance
	DataSchema
	Observable bool `json:"observable,omitempty"`
}

/*
An Interaction Affordance that allows to invoke a function of the Thing, which manipulates state (e.g., toggling a lamp on or off) or triggers a process on the Thing (e.g., dim a lamp over time).
*/
type ActionAffordance struct {
	InteractionAffordance

	// Used to define the input data schema of the Action.
	Input DataSchema `json:"input,omitempty"`

	// Used to define the output data schema of the Action.
	Output DataSchema `json:"input,omitempty"`

	// Signals if the Action is safe (=true) or not. Used to signal if there is no internal state (cf. resource state) is changed when invoking an Action. In that case responses can be cached as example.
	Safe bool `json:"safe"` //default: false

	// Indicates whether the Action is idempotent (=true) or not. Informs whether the Action can be called repeatedly with the same result, if present, based on the same input.
	Idempotent bool `json:"idempotent"` //default: false
}

/*
An Interaction Affordance that describes an event source, which asynchronously pushes event data to Consumers (e.g., overheating alerts).
*/
type EventAffordance struct {
	InteractionAffordance

	// Defines data that needs to be passed upon subscription, e.g., filters or message format for setting up Webhooks.
	Subscription DataSchema `json:"subscription,omitempty"`

	// Defines the data schema of the Event instance messages pushed by the Thing.
	Data DataSchema `json:"data,omitempty"`

	// Defines any data that needs to be passed to cancel a subscription, e.g., a specific message to remove a Webhook.
	Cancellation DataSchema `json:"optional,omitempty"`
}

/*
A form can be viewed as a statement of "To perform an operation type operation on form context, make a request method request to submission target" where the optional form fields may further describe the required request.
In Thing Descriptions, the form context is the surrounding Object, such as Properties, Actions, and Events or the Thing itself for meta-interactions.
*/
type Form struct {

	/*
		Indicates the semantic intention of performing the operation(s) described by the form.
		For example, the Property interaction allows get and set operations.
		The protocol binding may contain a form for the get operation and a different form for the set operation.
		The op attribute indicates which form is for which and allows the client to select the correct form for the operation required.
		op can be assigned one or more interaction verb(s) each representing a semantic intention of an operation.
		It can be one of: readproperty, writeproperty, observeproperty, unobserveproperty, invokeaction, subscribeevent, unsubscribeevent, readallproperties, writeallproperties, readmultipleproperties, or writemultipleproperties
		a. When a Form instance is within an ActionAffordance instance, the value assigned to op MUST be invokeaction.
		b. When a Form instance is within an EventAffordance instance, the value assigned to op MUST be either subscribeevent, unsubscribeevent, or both terms within an Array.
		c. When a Form instance is within a PropertyAffordance instance, the value assigned to op MUST be one of readproperty, writeproperty, observeproperty, unobserveproperty or an Array containing a combination of these terms.
	*/
	Op any `json:"op"`

	// Target IRI of a link or submission target of a form.
	Href AnyURI `json:"href"`

	// Assign a content type based on a media type (e.g., text/plain) and potential parameters (e.g., charset=utf-8) for the media type [RFC2046].
	ContentType string `json:"contentType"` //default: "application/json"

	// Content coding values indicate an encoding transformation that has been or can be applied to a representation. Content codings are primarily used to allow a representation to be compressed or otherwise usefully transformed without losing the identity of its underlying media type and without loss of information. Examples of content coding include "gzip", "deflate", etc. .
	// Possible values for the contentCoding property can be found, e.g., in thttps://www.iana.org/assignments/http-parameters/http-parameters.xhtml#content-coding
	ContentCoding string `json:"contentCoding,omitempty"`

	// Indicates the exact mechanism by which an interaction will be accomplished for a given protocol when there are multiple options.
	// For example, for HTTP and Events, it indicates which of several available mechanisms should be used for asynchronous notifications such as long polling (longpoll), WebSub [websub] (websub), Server-Sent Events [eventsource] (sse). Please note that there is no restriction on the subprotocol selection and other mechanisms can also be announced by this subprotocol term.
	SubProtocol string `json:"subprotocol,omitempty"`

	// Set of security definition names, chosen from those defined in securityDefinitions. These must all be satisfied for access to resources.
	Security any `json:"security,omitempty"`

	// Set of authorization scope identifiers provided as an array. These are provided in tokens returned by an authorization server and associated with forms in order to identify what resources a client may access and how. The values associated with a form should be chosen from those defined in an OAuth2SecurityScheme active on that form.
	Scopes any `json:"scopes,omitempty"`

	// This optional term can be used if, e.g., the output communication metadata differ from input metadata (e.g., output contentType differ from the input contentType). The response name contains metadata that is only valid for the response messages.
	Response *ExpectedResponse `json:"response,omitempty"`
}

/*
A link can be viewed as a statement of the form "link context has a relation type resource at link target", where the optional target attributes may further describe the resource.
*/
type Link struct {
	// Target IRI of a link or submission target of a form.
	Href AnyURI `json:"href"`

	// Target attribute providing a hint indicating what the media type (RFC2046) of the result of dereferencing the link should be.
	Type string `json:"type,omitempty"`

	// A link relation type identifies the semantics of a link.
	Rel string `json:"type,omitempty"`

	// Overrides the link context (by default the Thing itself identified by its id) with the given URI or IRI.
	Anchor AnyURI `json:"anchor,omitempty"`
}

type SecurityScheme struct {
	// JSON-LD keyword to label the object with semantic tags (or types).
	Type any `json:"@type,omitempty"`

	// Identification of the security mechanism being configured. e.g. nosec, basic, cert, digest, bearer, pop, psk, public, oauth2, or apike
	Scheme string `json:"scheme"`

	// Provides additional (human-readable) information based on a default language
	Description string `json:"description,omitempty"`

	// Can be used to support (human-readable) information in different languages.
	Descriptions map[string]string `json:"descriptions,omitempty"`

	// URI of the proxy server this security configuration provides access to. If not given, the corresponding security configuration is for the endpoint.
	Proxy AnyURI `json:"proxy,omitempty"`

	*BasicSecurityScheme
	*DigestSecurityScheme
	*APIKeySecurityScheme
	*BearerSecurityScheme
	*CertSecurityScheme
	*PSKSecurityScheme
	*PublicSecurityScheme
	*PoPSecurityScheme
	*OAuth2SecurityScheme
}

type DataSchema struct {
	// TJSON-LD keyword to label the object with semantic tags (or types)
	Type any `json:"@type,omitempty"`

	// Const corresponds to the JSON schema field "const".
	Const any `json:"const,omitempty"`

	// Provides multi-language human-readable titles (e.g., display a text for UI representation in different languages).
	Description string `json:"description,omitempty"`

	// Can be used to support (human-readable) information in different languages
	Descriptions string `json:"descriptions,omitempty"`

	// Restricted set of values provided as an array.
	Enum []any `json:"enum,omitempty"`

	// Allows validation based on a format pattern such as "date-time", "email", "uri", etc. (Also see below.)
	Format string `json:"format,omitempty"`

	// OneOf corresponds to the JSON schema field "oneOf".
	OneOf []DataSchema `json:"oneOf,omitempty"`

	// ReadOnly corresponds to the JSON schema field "readOnly".
	ReadOnly bool `json:"readOnly,omitempty"`

	// Provides a human-readable title (e.g., display a text for UI representation) based on a default language.
	Title string `json:"title,omitempty"`

	// Provides multi-language human-readable titles (e.g., display a text for UI representation in different languages).
	Titles []string `json:"titles,omitempty"`

	// Type_2 corresponds to the JSON schema field "type".
	DataType string `json:"type,omitempty"`

	// Unit corresponds to the JSON schema field "unit".
	Unit string `json:"unit,omitempty"`

	// Boolean value that is a hint to indicate whether a property interaction / value is write only (=true) or not (=false).
	WriteOnly bool `json:"writeOnly,omitempty"`

	// Metadata describing data of type Array. This Subclass is indicated by the value array assigned to type in DataSchema instances.
	*ArraySchema

	// Metadata describing data of type number. This Subclass is indicated by the value number assigned to type in DataSchema instances.
	*NumberSchema

	// Metadata describing data of type object. This Subclass is indicated by the value object assigned to type in DataSchema instances.
	*ObjectSchema
}

type ArraySchema struct {
	// Used to define the characteristics of an array.
	Items any `json:"items,omitempty"`

	// Defines the maximum number of items that have to be in the array.
	MaxItems *int `json:"maxItems,omitempty"`

	// Defines the minimum number of items that have to be in the array.
	MinItems *int `json:"minItems,omitempty"`
}

//Specifies both float and double
type NumberSchema struct {
	// Specifies a maximum numeric value. Only applicable for associated number or integer types.
	Maximum *any `json:"maximum,omitempty"`

	// Specifies a minimum numeric value. Only applicable for associated number or integer types.
	Minimum *any `json:"minimum,omitempty"`
}

type ObjectSchema struct {
	// Data schema nested definitions.
	Properties map[string]DataSchema `json:"properties,omitempty"`

	// Required corresponds to the JSON schema field "required".
	Required []string `json:"required,omitempty"`
}

type AnyURI = string

/*
Communication metadata describing the expected response message.
*/
type ExpectedResponse struct {
	ContentType string `json:"contentType,omitempty"`
}

/*
Metadata of a Thing that provides version information about the TD document. If required, additional version information such as firmware and hardware version (term definitions outside of the TD namespace) can be extended via the TD Context Extension mechanism.
It is recommended that the values within instances of the VersionInfo Class follow the semantic versioning pattern (https://semver.org/), where a sequence of three numbers separated by a dot indicates the major version, minor version, and patch version, respectively.
*/
type VersionInfo struct {
	// Provides a version indicator of this TD instance.
	Instance string `json:"instance"`
}

/*
Basic Authentication [RFC7617] security configuration identified by the Vocabulary Term basic (i.e., "scheme": "basic"), using an unencrypted username and password.
This scheme should be used with some other security mechanism providing confidentiality, for example, TLS.
*/
type BasicSecurityScheme struct {
	// Specifies the location of security authentication information.
	In string `json:"in"` // default: header

	// Name for query, header, or cookie parameters.
	Name string `json:"name,omitempty"`
}

/*
Digest Access Authentication [RFC7616] security configuration identified by the Vocabulary Term digest (i.e., "scheme": "digest").
This scheme is similar to basic authentication but with added features to avoid man-in-the-middle attacks.
*/
type DigestSecurityScheme struct {
	// Specifies the location of security authentication information.
	In string `json:"in"` // default: header

	// Name for query, header, or cookie parameters.
	Name string `json:"name,omitempty"`

	//Quality of protection.
	Qop string `json:"qop,omitempty"` //default: auth
}

/*
API key authentication security configuration identified by the Vocabulary Term apikey (i.e., "scheme": "apikey").
This is for the case where the access token is opaque and is not using a standard token format.
*/
type APIKeySecurityScheme struct {
	// Specifies the location of security authentication information.
	In string `json:"in"` // default: header

	// Name for query, header, or cookie parameters.
	Name string `json:"name,omitempty"`
}

/*
Bearer Token [RFC6750] security configuration identified by the Vocabulary Term bearer (i.e., "scheme": "bearer") for situations where bearer tokens are used independently of OAuth2. If the oauth2 scheme is specified it is not generally necessary to specify this scheme as well as it is implied. For format, the value jwt indicates conformance with [RFC7519], jws indicates conformance with [RFC7797], cwt indicates conformance with [RFC8392], and jwe indicates conformance with [RFC7516], with values for alg interpreted consistently with those standards.
Other formats and algorithms for bearer tokens MAY be specified in vocabulary extensions
*/
type BearerSecurityScheme struct {
	// Specifies the location of security authentication information.
	In string `json:"in"` // default: header

	// Name for query, header, or cookie parameters.
	Name string `json:"name,omitempty"`

	// URI of the authorization server.
	Authorization AnyURI `json:"authorization,omitempty"`

	// Encoding, encryption, or digest algorithm.
	Alg string `json:"alg"` // default:ES256

	// Specifies format of security authentication information.
	Format string `json:"format"` // default: jwt
}

/*
Certificate-based asymmetric key security configuration conformant with [X509V3] identified by the Vocabulary Term cert (i.e., "scheme": "cert").
*/
type CertSecurityScheme struct {
	// Identifier providing information which can be used for selection or confirmation.
	Identity string `json:"identity,omitempty"`
}

/*
Pre-shared key authentication security configuration identified by the Vocabulary Term psk (i.e., "scheme": "psk").
*/
type PSKSecurityScheme struct {
	// Identifier providing information which can be used for selection or confirmation.
	Identity string `json:"identity,omitempty"`
}

/*
Raw public key asymmetric key security configuration identified by the Vocabulary Term public (i.e., "scheme": "public").
*/
type PublicSecurityScheme struct {
	// Identifier providing information which can be used for selection or confirmation.
	Identity string `json:"identity,omitempty"`
}

/*
Proof-of-possession (PoP) token authentication security configuration identified by the Vocabulary Term pop (i.e., "scheme": "pop"). Here jwt indicates conformance with [RFC7519], jws indicates conformance with [RFC7797], cwt indicates conformance with [RFC8392], and jwe indicates conformance with [RFC7516], with values for alg interpreted consistently with those standards.
Other formats and algorithms for PoP tokens MAY be specified in vocabulary extensions..
*/
type PoPSecurityScheme struct {
	// Specifies the location of security authentication information.
	In string `json:"in"` // default: header

	// Name for query, header, or cookie parameters.
	Name string `json:"name,omitempty"`

	// Encoding, encryption, or digest algorithm.
	Alg string `json:"alg"` // default:ES256

	// Specifies format of security authentication information.
	Format string `json:"format"` // default: jwt

	// URI of the authorization server.
	Authorization AnyURI `json:"authorization,omitempty"`
}

/*
OAuth2 authentication security configuration for systems conformant with [RFC6749] and [RFC8252], identified by the Vocabulary Term oauth2 (i.e., "scheme": "oauth2").
For the implicit flow authorization MUST be included. For the password and client flows token MUST be included.
For the code flow both authorization and token MUST be included. If no scopes are defined in the SecurityScheme then they are considered to be empty.
*/
type OAuth2SecurityScheme struct {
	// URI of the authorization server.
	Authorization AnyURI `json:"authorization,omitempty"`

	//URI of the token server.
	Token AnyURI `json:"token,omitempty"`

	//URI of the refresh server.
	Refresh AnyURI `json:"refresh,omitempty"`

	//Set of authorization scope identifiers provided as an array. These are provided in tokens returned by an authorization server and associated with forms in order to identify what resources a client may access and how. The values associated with a form should be chosen from those defined in an OAuth2SecurityScheme active on that form.
	Scopes any `json:"scopes,omitempty"`

	//Authorization flow.
	Flow string `json:"flow"`
}

var enumValues_DataSchemaType = []any{
	"boolean",
	"integer",
	"number",
	"string",
	"object",
	"array",
	"null",
}
