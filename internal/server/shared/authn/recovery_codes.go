package authn

import "pass-pivot/utils"

func RecoveryCodes() []string {
	codes := make([]string, 0, 10)
	for i := 0; i < 10; i++ {
		codes = append(codes, utils.RandomHumanToken(5))
	}
	return codes
}
