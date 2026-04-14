// SPDX-License-Identifier: MIT

package core

// Standard validation error reasons.
const (
	// String/Collection reasons
	ReasonMinLength    = "length is less than required minimum"
	ReasonMaxLength    = "length exceeds maximum limit"
	ReasonPattern      = "value does not match required pattern"
	ReasonInvalidEmail = "invalid email format"
	ReasonInvalidUUID  = "invalid UUID format"
	ReasonInvalidURL   = "invalid URL format"
	ReasonInvalidIP    = "invalid IP address"
	ReasonInvalidIPv4  = "invalid IPv4 address"
	ReasonInvalidIPv6  = "invalid IPv6 address"

	// Numeric/Value reasons
	ReasonMinValue = "value is less than minimum"
	ReasonMaxValue = "value exceeds maximum limit"

	// Required/Presence reasons
	ReasonRequired = "value is required"
)
