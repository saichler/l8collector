/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ifs

import (
	"github.com/saichler/l8types/go/types/l8sysconfig"
	"net"
)

// ISecurityProvider defines the security interface for authentication, authorization,
// encryption, and connection validation in the Layer 8 system.
type ISecurityProvider interface {
	// Authenticate validates user credentials and returns a token.
	// Returns: token, needsTFASetup, needsTFAVerify, error
	Authenticate(string, string, IVNic) (string, bool, bool, error)
	// ValidateToken checks if a token is valid and returns the user ID.
	ValidateToken(string, IVNic) (string, bool)

	// Message creates a message from an AAA ID for audit/tracking.
	Message(string, IVNic) (*Message, error)

	// CanDial validates if a connection to the address:port is allowed.
	CanDial(string, uint32) (net.Conn, error)
	// CanAccept validates if an incoming connection should be accepted.
	CanAccept(net.Conn) error
	// ValidateConnection performs full connection validation with config.
	ValidateConnection(net.Conn, *l8sysconfig.L8SysConfig) error

	// Encrypt encrypts data bytes to a string (typically base64).
	Encrypt([]byte) (string, error)
	// Decrypt decrypts a string back to data bytes.
	Decrypt(string) ([]byte, error)

	// CanDoAction checks if an action is authorized for the given user/scope.
	CanDoAction(IVNic, Action, IElements, string, string, ...string) error
	// ScopeView filters elements based on user permissions.
	ScopeView(IElements, string, string, ...string) IElements

	// TFASetup initiates two-factor authentication setup.
	// Returns: secret, QR code bytes, error
	TFASetup(string, IVNic) (string, []byte, error)
	// TFAVerify validates a TFA code.
	TFAVerify(string, string, string, IVNic) error

	// Captcha generates a captcha challenge.
	Captcha() []byte
	// Register creates a new user account.
	Register(string, string, string, IVNic) error

	// Credential retrieves credential components by name and type.
	// Returns: aside, zside, yside, name, error
	Credential(string, string, IResources) (string, string, string, string, error)
}

// ISecurityProviderLoader loads security provider plugins.
type ISecurityProviderLoader interface {
	// LoadSecurityProvider loads and initializes a security provider.
	LoadSecurityProvider(...interface{}) (ISecurityProvider, error)
}

// ISecurityProviderActivate provides activation lifecycle for security providers.
type ISecurityProviderActivate interface {
	// Activate initializes the security provider with a VNic.
	Activate(IVNic)
}
