// +build cgo

package snmp

/*
#cgo CFLAGS: -I/usr/include
#cgo LDFLAGS: -lnetsnmp

#include <stdlib.h>
#include <string.h>
#include <net-snmp/net-snmp-config.h>
#include <net-snmp/net-snmp-includes.h>

// Global flag to track if SNMP library is initialized
static int snmp_initialized = 0;

// Helper function to initialize SNMP library
void snmp_init() {
    if (!snmp_initialized) {
        init_snmp("l8collector");
        snmp_initialized = 1;
    }
}

// Helper function to validate session
int validate_session(netsnmp_session* session) {
    if (!session) return 0;
    if (session->version < 0 || session->version > 3) return 0;
    if (!session->peername) return 0;
    return 1;
}

// Helper function to create a session
netsnmp_session* create_snmp_session(char* host, char* community) {
    if (!host || !community || strlen(host) == 0 || strlen(community) == 0) return NULL;

    netsnmp_session session, *ss;

    // Initialize session structure
    snmp_sess_init(&session);

    // Set session parameters
    session.peername = host;
    session.version = SNMP_VERSION_2c;
    session.community = (u_char*)community;
    session.community_len = strlen(community);

    // Set reasonable timeouts
    session.timeout = 1000000; // 1 second in microseconds
    session.retries = 3;

    ss = snmp_open(&session);
    return ss;
}

// Helper function to perform SNMP walk
int snmp_walk_helper(netsnmp_session* session, char* oid_str, char** result_json) {
    if (!session || !oid_str || !result_json) return -1;

    // Initialize result pointer
    *result_json = NULL;

    // Validate session pointer more thoroughly
    if (!validate_session(session)) {
        return -6; // Invalid session
    }

    oid name[MAX_OID_LEN];
    oid root_oid[MAX_OID_LEN]; // Preserve original OID for subtree checking
    size_t name_len = MAX_OID_LEN;
    size_t root_oid_len;

    // Clear the OID arrays
    memset(name, 0, sizeof(name));
    memset(root_oid, 0, sizeof(root_oid));

    if (!snmp_parse_oid(oid_str, name, &name_len)) {
        return -2; // OID parse error
    }

    // Save the original OID for subtree boundary checking
    memmove(root_oid, name, name_len * sizeof(oid));
    root_oid_len = name_len;

    netsnmp_pdu *pdu = NULL, *response = NULL;
    netsnmp_variable_list *vars;
    int status;
    int count = 0;
    int max_iterations = 1000; // Reduced from 10000 to be more conservative
    size_t buffer_size = 65536; // Start with 64KB buffer (reduced from 256KB)
    char *json_result = malloc(buffer_size);
    if (!json_result) return -3; // Memory allocation error

    // Initialize buffer safely
    memset(json_result, 0, buffer_size);
    size_t current_pos = 0;
    json_result[0] = '[';
    json_result[1] = '\0';
    current_pos = 1;

    pdu = snmp_pdu_create(SNMP_MSG_GETNEXT);
    if (!pdu) {
        free(json_result);
        return -5; // PDU creation error
    }
    snmp_add_null_var(pdu, name, name_len);

    while (count < max_iterations) {
        status = snmp_synch_response(session, pdu, &response);

        if (status == STAT_SUCCESS && response && response->errstat == SNMP_ERR_NOERROR) {
            for (vars = response->variables; vars; vars = vars->next_variable) {
                // Additional safety checks for vars pointer
                if (!vars || !vars->name || vars->name_length == 0 || vars->name_length > MAX_OID_LEN) {
                    continue;
                }

                // Check if we've walked past our subtree
                // The returned OID must be within the original requested subtree
                if (vars->name_length < root_oid_len ||
                    snmp_oid_compare(root_oid, root_oid_len, vars->name, root_oid_len) != 0) {
                    goto done; // Walked past our subtree
                }

                // Check if this is the end of MIB view or no such object
                if (vars->type == SNMP_ENDOFMIBVIEW || vars->type == SNMP_NOSUCHOBJECT || vars->type == SNMP_NOSUCHINSTANCE) {
                    goto done; // End of MIB walk
                }

                // Check if we got the same OID as before (infinite loop detection)
                if (snmp_oid_compare(name, name_len, vars->name, vars->name_length) == 0) {
                    goto done; // Same OID returned, stop to prevent infinite loop
                }

                // Convert OID to string
                char oid_buf[512];
                memset(oid_buf, 0, sizeof(oid_buf));
                if (snprint_objid(oid_buf, sizeof(oid_buf)-1, vars->name, vars->name_length) <= 0) {
                    continue; // Skip if OID conversion fails
                }

                // Convert value to string - escape quotes and special chars
                char val_buf[1024];
                memset(val_buf, 0, sizeof(val_buf));
                if (snprint_value(val_buf, sizeof(val_buf)-1, vars->name, vars->name_length, vars) <= 0) {
                    strcpy(val_buf, ""); // Use empty string if value conversion fails
                }

                // Check for end-of-MIB indicators in the value string
                if (strstr(val_buf, "No more variables left") != NULL ||
                    strstr(val_buf, "End of MIB") != NULL ||
                    strstr(val_buf, "past the end of the MIB tree") != NULL) {
                    goto done; // End of MIB walk detected in value
                }

                // Escape special characters in value for JSON with safer bounds checking
                char escaped_val[2048];  // Reduced size to be more conservative
                memset(escaped_val, 0, sizeof(escaped_val));
                size_t j = 0;
                size_t val_len = strlen(val_buf);
                size_t max_escaped = sizeof(escaped_val) - 10; // Leave safety margin

                for (size_t i = 0; i < val_len && j < max_escaped; i++) {
                    char c = val_buf[i];
                    if (j >= max_escaped - 6) break; // Ensure we have room for escape sequences

                    if (c == '"' || c == '\\') {
                        if (j < max_escaped - 1) {
                            escaped_val[j++] = '\\';
                            escaped_val[j++] = c;
                        }
                    } else if (c == '\n') {
                        if (j < max_escaped - 1) {
                            escaped_val[j++] = '\\';
                            escaped_val[j++] = 'n';
                        }
                    } else if (c == '\r') {
                        if (j < max_escaped - 1) {
                            escaped_val[j++] = '\\';
                            escaped_val[j++] = 'r';
                        }
                    } else if (c == '\t') {
                        if (j < max_escaped - 1) {
                            escaped_val[j++] = '\\';
                            escaped_val[j++] = 't';
                        }
                    } else if ((unsigned char)c < 32) {
                        // Escape other control characters as \uXXXX
                        if (j < max_escaped - 6) {
                            int written = snprintf(escaped_val + j, max_escaped - j, "\\u%04x", (unsigned char)c);
                            if (written > 0 && written < (int)(max_escaped - j)) {
                                j += written;
                            }
                        }
                    } else {
                        escaped_val[j++] = c;
                    }
                }
                escaped_val[j] = '\0'; // Ensure null termination

                // Calculate needed space for this entry
                size_t entry_needed = strlen(oid_buf) + strlen(escaped_val) + 50; // Extra space for JSON formatting

                // Check if we need to grow the buffer
                if (current_pos + entry_needed + 10 > buffer_size) {
                    size_t new_buffer_size = buffer_size * 2;
                    char *new_buffer = realloc(json_result, new_buffer_size);
                    if (!new_buffer) {
                        free(json_result);
                        if (response) snmp_free_pdu(response);
                        return -4; // Memory reallocation error
                    }
                    json_result = new_buffer;
                    buffer_size = new_buffer_size;
                }

                // Add comma if not first entry
                if (count > 0) {
                    json_result[current_pos++] = ',';
                }

                // Add JSON entry safely
                int written = snprintf(json_result + current_pos, buffer_size - current_pos - 10,
                                     "{\"oid\":\"%s\",\"value\":\"%s\"}", oid_buf, escaped_val);
                if (written > 0 && written < (int)(buffer_size - current_pos - 10)) {
                    current_pos += written;
                    count++;
                }

                // Setup for next request
                if (vars->name_length <= MAX_OID_LEN) {
                    memmove(name, vars->name, vars->name_length * sizeof(oid));
                    name_len = vars->name_length;
                } else {
                    goto done; // OID too long, stop walking
                }
            }

            if (response) {
                snmp_free_pdu(response);
                response = NULL;
            }

            pdu = snmp_pdu_create(SNMP_MSG_GETNEXT);
            if (!pdu) {
                break; // Can't create PDU, stop walking
            }
            snmp_add_null_var(pdu, name, name_len);
        } else {
            if (response) {
                snmp_free_pdu(response);
                response = NULL;
            }
            break;
        }
    }

done:
    // Safely close JSON array
    if (current_pos < buffer_size - 2) {
        json_result[current_pos++] = ']';
        json_result[current_pos] = '\0';
    }

    *result_json = json_result;

    if (response) snmp_free_pdu(response);
    return count;
}

// Helper function to clean up session
void close_snmp_session(netsnmp_session* session) {
    if (session) {
        snmp_close(session);
    }
}
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"sync"
	"unsafe"
)

// Global initialization mutex to ensure SNMP library is initialized only once
// Also used to serialize all SNMP operations due to thread safety issues in net-snmp
var (
	initMutex    sync.Mutex
	initDone     bool
	globalSNMPMutex sync.Mutex // Serialize all SNMP operations globally
)

// SNMPSession represents a net-snmp session
type SNMPSession struct {
	session   unsafe.Pointer
	host      string
	community string
	mutex     sync.Mutex // Protect concurrent access to session
}

// snmpResult represents a single SNMP result entry
type snmpResult struct {
	OID   string `json:"oid"`
	Value string `json:"value"`
}

// initSNMP ensures SNMP library is initialized exactly once
func initSNMP() {
	initMutex.Lock()
	defer initMutex.Unlock()

	if !initDone {
		C.snmp_init()
		initDone = true
	}
}

// NewSNMPSession creates a new SNMP session using net-snmp library
func NewSNMPSession(host, community string) (*SNMPSession, error) {
	// Serialize session creation due to net-snmp thread safety issues
	globalSNMPMutex.Lock()
	defer globalSNMPMutex.Unlock()

	// Validate input parameters
	if host == "" || community == "" {
		return nil, fmt.Errorf("host and community cannot be empty")
	}

	// Initialize SNMP library exactly once
	initSNMP()

	hostCStr := C.CString(host)
	defer C.free(unsafe.Pointer(hostCStr))

	communityCStr := C.CString(community)
	defer C.free(unsafe.Pointer(communityCStr))

	session := C.create_snmp_session(hostCStr, communityCStr)
	if session == nil {
		return nil, fmt.Errorf("failed to create SNMP session for host %s", host)
	}

	return &SNMPSession{
		session: unsafe.Pointer(session),
		host: host,
		community: community,
	}, nil
}

// Walk performs an SNMP walk operation
func (s *SNMPSession) Walk(oid string) ([]SnmpPDU, error) {
	if s == nil {
		return nil, fmt.Errorf("session is nil")
	}

	// Serialize all SNMP operations globally due to net-snmp thread safety issues
	globalSNMPMutex.Lock()
	defer globalSNMPMutex.Unlock()

	// Also lock the session for additional safety
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.session == nil {
		return nil, fmt.Errorf("session is not initialized")
	}

	if oid == "" {
		return nil, fmt.Errorf("OID cannot be empty")
	}

	oidCStr := C.CString(oid)
	defer C.free(unsafe.Pointer(oidCStr))

	var resultCStr *C.char
	count := C.snmp_walk_helper((*C.netsnmp_session)(s.session), oidCStr, &resultCStr)

	// Handle error codes
	switch count {
	case -1:
		return nil, fmt.Errorf("SNMP walk failed: invalid session or parameters")
	case -2:
		return nil, fmt.Errorf("SNMP walk failed: invalid OID '%s'", oid)
	case -3:
		return nil, fmt.Errorf("SNMP walk failed: memory allocation error")
	case -4:
		return nil, fmt.Errorf("SNMP walk failed: memory reallocation error")
	case -5:
		return nil, fmt.Errorf("SNMP walk failed: PDU creation error")
	case -6:
		return nil, fmt.Errorf("SNMP walk failed: invalid session version")
	}

	if count < 0 {
		return nil, fmt.Errorf("SNMP walk failed with code %d", count)
	}

	if resultCStr == nil || count == 0 {
		return []SnmpPDU{}, nil
	}
	defer C.free(unsafe.Pointer(resultCStr))

	// Parse JSON result from C
	jsonStr := C.GoString(resultCStr)
	if jsonStr == "" || jsonStr == "[]" {
		return []SnmpPDU{}, nil
	}

	var results []snmpResult
	if err := json.Unmarshal([]byte(jsonStr), &results); err != nil {
		return nil, fmt.Errorf("failed to parse SNMP results: %v (JSON: %s)", err, jsonStr)
	}

	// Convert to SnmpPDU format
	var pdus []SnmpPDU
	for _, result := range results {
		pdus = append(pdus, SnmpPDU{
			Name:  result.OID,
			Value: result.Value,
		})
	}

	return pdus, nil
}

// Close closes the SNMP session
func (s *SNMPSession) Close() error {
	if s == nil {
		return nil // Closing a nil session is a no-op
	}

	// Serialize close operations as well
	globalSNMPMutex.Lock()
	defer globalSNMPMutex.Unlock()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.session != nil {
		C.close_snmp_session((*C.netsnmp_session)(s.session))
		s.session = nil
	}
	return nil
}