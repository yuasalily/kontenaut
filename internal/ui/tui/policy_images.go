package tui

// Image action policy (minimal, initial).
//
// Why a dedicated policy file:
// - Images may gain more actions in the future (prune, tag, inspect, etc).
// - Keep action availability decisions in one place for consistency and expansion.

// canDeleteImage returns whether the given image ID can be deleted by policy.
//
// Minimal policy (initial):
// - referenced by any container -> delete not allowed
// - otherwise -> delete allowed
func canDeleteImage(imageID string, nonDeletable map[string]struct{}) bool {
	if imageID == "" {
		return false
	}
	_, blocked := nonDeletable[imageID]
	return !blocked
}

// nonDeletableImageIDs returns IDs of images that cannot be deleted by policy.
//
// At the moment the policy is identical to "referenced by any container".
// Keeping this function allows future expansion without touching pages.
func nonDeletableImageIDs(referenced map[string]struct{}) map[string]struct{} {
	if referenced == nil {
		return map[string]struct{}{}
	}
	return referenced
}
