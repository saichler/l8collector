// +build cgo

package snmp

/*
#cgo CFLAGS: -I/usr/include
#cgo LDFLAGS: -lnetsnmp

#include <stdlib.h>
#include <string.h>
#include <net-snmp/net-snmp-config.h>
#include <net-snmp/net-snmp-includes.h>

// Helper function to initialize SNMP library
void snmp_init() {
    init_snmp("l8collector");
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

    oid name[MAX_OID_LEN];
    size_t name_len = MAX_OID_LEN;

    // Clear the OID array
    memset(name, 0, sizeof(name));

    if (!snmp_parse_oid(oid_str, name, &name_len)) {
        return -2; // OID parse error
    }

    netsnmp_pdu *pdu = NULL, *response = NULL;
    netsnmp_variable_list *vars;
    int status;
    int count = 0;
    int max_iterations = 10000; // Prevent infinite loops
    size_t buffer_size = 262144; // Start with 256KB buffer
    char *json_result = malloc(buffer_size);
    if (!json_result) return -3; // Memory allocation error

    // Initialize buffer
    memset(json_result, 0, buffer_size);
    size_t current_pos = 0;
    strcpy(json_result, "[");
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
                // Check if we've walked past our subtree
                if (snmp_oid_compare(name, name_len, vars->name, vars->name_length) > 0) {
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

                // Null check for variable data
                if (!vars->name || vars->name_length == 0) {
                    continue;
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

                // Escape special characters in value for JSON
                char escaped_val[4096];  // Increased size for escaped content
                memset(escaped_val, 0, sizeof(escaped_val));
                int j = 0;
                size_t val_len = strlen(val_buf);
                for (size_t i = 0; i < val_len && j < sizeof(escaped_val)-3; i++) {
                    char c = val_buf[i];
                    if (c == '"' || c == '\\') {
                        escaped_val[j++] = '\\';
                        escaped_val[j++] = c;
                    } else if (c == '\n') {
                        escaped_val[j++] = '\\';
                        escaped_val[j++] = 'n';
                    } else if (c == '\r') {
                        escaped_val[j++] = '\\';
                        escaped_val[j++] = 'r';
                    } else if (c == '\t') {
                        escaped_val[j++] = '\\';
                        escaped_val[j++] = 't';
                    } else if (c == '\b') {
                        escaped_val[j++] = '\\';
                        escaped_val[j++] = 'b';
                    } else if (c == '\f') {
                        escaped_val[j++] = '\\';
                        escaped_val[j++] = 'f';
                    } else if ((unsigned char)c < 32) {
                        // Escape other control characters as \uXXXX
                        j += snprintf(escaped_val + j, sizeof(escaped_val) - j - 1, "\\u%04x", (unsigned char)c);
                    } else {
                        escaped_val[j++] = c;
                    }
                }

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

// NewSNMPSession creates a new SNMP session using net-snmp library
func NewSNMPSession(host, community string) (*SNMPSession, error) {
	// Initialize SNMP library
	C.snmp_init()

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

	// Lock the session for thread safety
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

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.session != nil {
		C.close_snmp_session((*C.netsnmp_session)(s.session))
		s.session = nil
	}
	return nil
}