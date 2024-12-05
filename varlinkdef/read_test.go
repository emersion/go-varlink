package varlinkdef_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/emersion/go-varlink/varlinkdef"
)

var serviceRaw = `# The Varlink Service Interface is provided by every varlink service. It
# describes the service and the interfaces it implements.
interface org.varlink.service

# Get a list of all the interfaces a service provides and information
# about the implementation.
method GetInfo() -> (
  vendor: string,
  product: string,
  version: string,
  url: string,
  interfaces: []string
)

# Get the description of an interface that is implemented by this service.
method GetInterfaceDescription(interface: string) -> (description: string)

# The requested interface was not found.
error InterfaceNotFound (interface: string)

# The requested method was not found
error MethodNotFound (method: string)

# The interface defines the requested method, but the service does not
# implement it.
error MethodNotImplemented (method: string)

# One of the passed parameters is invalid.
error InvalidParameter (parameter: string)
`

var serviceIface = &varlinkdef.Interface{
	Name:  "org.varlink.service",
	Types: map[string]varlinkdef.Type{},
	Methods: map[string]varlinkdef.Method{
		"GetInfo": varlinkdef.Method{
			In: varlinkdef.Struct{},
			Out: varlinkdef.Struct{
				"vendor":  varlinkdef.TypeString,
				"product": varlinkdef.TypeString,
				"version": varlinkdef.TypeString,
				"url":     varlinkdef.TypeString,
				"interfaces": varlinkdef.Type{
					Kind:  varlinkdef.KindArray,
					Inner: &varlinkdef.TypeString,
				},
			},
		},
		"GetInterfaceDescription": varlinkdef.Method{
			In: varlinkdef.Struct{
				"interface": varlinkdef.TypeString,
			},
			Out: varlinkdef.Struct{
				"description": varlinkdef.TypeString,
			},
		},
	},
	Errors: map[string]varlinkdef.Struct{
		"InterfaceNotFound": varlinkdef.Struct{
			"interface": varlinkdef.TypeString,
		},
		"MethodNotFound": varlinkdef.Struct{
			"method": varlinkdef.TypeString,
		},
		"MethodNotImplemented": varlinkdef.Struct{
			"method": varlinkdef.TypeString,
		},
		"InvalidParameter": varlinkdef.Struct{
			"parameter": varlinkdef.TypeString,
		},
	},
}

const exampleRaw = `# Interface to jump a spacecraft to another point in space.
# The FTL Drive is the propulsion system to achieve
# faster-than-light travel through space. A ship making a
# properly calculated jump can arrive safely in planetary
# orbit, or alongside other ships or spaceborne objects.
interface org.example.ftl

# The current state of the FTL drive and the amount of
# fuel available to jump.
type DriveCondition (
  state: (idle, spooling, busy),
  tylium_level: int
)

# Speed, trajectory and jump duration is calculated prior
# to activating the FTL drive.
type DriveConfiguration (
  speed: int,
  trajectory: int,
  duration: int
)

# The galactic coordinates use the Sun as the origin.
# Galactic longitude is measured with primary direction
# from the Sun to the center of the galaxy in the galactic
# plane, while the galactic latitude measures the angle
# of the object above the galactic plane.
type Coordinate (
  longitude: float,
  latitude: float,
  distance: int
)

# Monitor the drive. The method will reply with an update
# whenever the drive's state changes
method Monitor() -> (condition: DriveCondition)

# Calculate the drive's jump parameters from the current
# position to the target position in the galaxy
method CalculateConfiguration(
  current: Coordinate,
  target: Coordinate
) -> (configuration: DriveConfiguration)

# Jump to the calculated point in space
method Jump(configuration: DriveConfiguration) -> ()

# There is not enough tylium to jump with the given
# parameters
error NotEnoughEnergy ()

# The supplied parameters are outside the supported range
error ParameterOutOfRange (field: string)
`

var exampleIface = &varlinkdef.Interface{
	Name: "org.example.ftl",
	Types: map[string]varlinkdef.Type{
		"DriveCondition": varlinkdef.Type{
			Kind: varlinkdef.KindStruct,
			Struct: varlinkdef.Struct{
				"state": varlinkdef.Type{
					Kind: varlinkdef.KindEnum,
					Enum: varlinkdef.Enum{"idle", "spooling", "busy"},
				},
				"tylium_level": varlinkdef.TypeInt,
			},
		},
		"DriveConfiguration": varlinkdef.Type{
			Kind: varlinkdef.KindStruct,
			Struct: varlinkdef.Struct{
				"speed":      varlinkdef.TypeInt,
				"trajectory": varlinkdef.TypeInt,
				"duration":   varlinkdef.TypeInt,
			},
		},
		"Coordinate": varlinkdef.Type{
			Kind: varlinkdef.KindStruct,
			Struct: varlinkdef.Struct{
				"longitude": varlinkdef.TypeFloat,
				"latitude":  varlinkdef.TypeFloat,
				"distance":  varlinkdef.TypeInt,
			},
		},
	},
	Methods: map[string]varlinkdef.Method{
		"Monitor": varlinkdef.Method{
			In: varlinkdef.Struct{},
			Out: varlinkdef.Struct{
				"condition": varlinkdef.Type{
					Kind: varlinkdef.KindName,
					Name: "DriveCondition",
				},
			},
		},
		"CalculateConfiguration": varlinkdef.Method{
			In: varlinkdef.Struct{
				"current": varlinkdef.Type{
					Kind: varlinkdef.KindName,
					Name: "Coordinate",
				},
				"target": varlinkdef.Type{
					Kind: varlinkdef.KindName,
					Name: "Coordinate",
				},
			},
			Out: varlinkdef.Struct{
				"configuration": varlinkdef.Type{
					Kind: varlinkdef.KindName,
					Name: "DriveConfiguration",
				},
			},
		},
		"Jump": varlinkdef.Method{
			In: varlinkdef.Struct{
				"configuration": varlinkdef.Type{
					Kind: varlinkdef.KindName,
					Name: "DriveConfiguration",
				},
			},
			Out: varlinkdef.Struct{},
		},
	},
	Errors: map[string]varlinkdef.Struct{
		"NotEnoughEnergy": varlinkdef.Struct{},
		"ParameterOutOfRange": varlinkdef.Struct{
			"field": varlinkdef.TypeString,
		},
	},
}

func TestRead(t *testing.T) {
	tests := []struct {
		Name      string
		Raw       string
		Interface *varlinkdef.Interface
	}{
		{"org.varlink.service", serviceRaw, serviceIface},
		{"org.example.ftl", exampleRaw, exampleIface},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			r := strings.NewReader(tc.Raw)
			iface, err := varlinkdef.Read(r)
			if err != nil {
				t.Fatalf("Read() = %v", err)
			}
			if !reflect.DeepEqual(iface, tc.Interface) {
				t.Errorf("Read() = \n%#v\n but want \n%#v", iface, tc.Interface)
			}
		})
	}
}
