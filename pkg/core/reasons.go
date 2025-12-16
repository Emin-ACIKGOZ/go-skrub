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

	// Numeric/Value reasons (NEW)
	ReasonMinValue = "value is less than minimum"
	ReasonMaxValue = "value exceeds maximum limit"
)
