package errx

import "strings"

func DashScopeCodeToKind(code string) Kind {
	switch code {
	case "Arrearage":
		return KindUnauthorized
	case "InvalidParameter", "DataInspectionFailed":
		return KindInvalidArgument
	case "APIConnectionError":
		return KindNetwork
	case "AllocationQuota", "Throttling":
		return KindRateLimited
	default:
		if strings.HasPrefix(code, "Invalid") {
			return KindInvalidArgument
		}
		return KindInternal
	}
}

func MiniMaxCodeToKind(code int64) Kind {
	switch code {
	case 1001:
		return KindNetwork
	case 1002, 1041, 2045:
		return KindRateLimited
	case 1004, 1008, 2049:
		return KindUnauthorized
	case 1026, 1027, 1039, 1042, 2013, 2037, 2038, 2039, 2042, 2048:
		return KindInvalidArgument
	default:
		return KindInternal
	}
}

func OpenAIErrorTypeToKind(typ string) Kind {
	switch typ {
	case "invalid_request_error":
		return KindInvalidArgument
	case "authentication_error":
		return KindUnauthorized
	case "insufficient_quota", "rate_limit_error":
		return KindRateLimited
	case "server_error", "api_error":
		return KindInternal
	default:
		return KindInternal
	}
}
