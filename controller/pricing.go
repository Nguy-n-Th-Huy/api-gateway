package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

// filterPricingByUsableGroups keeps the model rows a requester at the given role
// may see, and prunes from enable_groups only the group names the admin-only rule
// withholds from that role. A model enabled for both an admin-only group and a
// usable group stays visible; only the withheld name disappears. Other names the
// requester cannot use are deliberately left alone — enable_groups has always
// been able to name them, and this change does not redefine that.
//
// The pruned row always carries a freshly allocated slice: pricing comes from
// model.GetPricing(), whose rows share each EnableGroup with the pricing cache
// (modelEnableGroups holds the same slice), so an in-place prune would leak this
// requester's view into every later response, including an administrator's.
func filterPricingByUsableGroups(pricing []model.Pricing, usableGroup map[string]string, role int) []model.Pricing {
	if len(pricing) == 0 {
		return pricing
	}
	if len(usableGroup) == 0 {
		return []model.Pricing{}
	}

	filtered := make([]model.Pricing, 0, len(pricing))
	for _, item := range pricing {
		visible := common.StringsContains(item.EnableGroup, "all")
		if !visible {
			for _, group := range item.EnableGroup {
				if _, ok := usableGroup[group]; ok {
					visible = true
					break
				}
			}
		}
		if !visible {
			continue
		}
		// "all" is a wildcard, not a group name, and must survive pruning.
		enableGroups := make([]string, 0, len(item.EnableGroup))
		for _, group := range item.EnableGroup {
			if group != "all" && service.IsAdminOnlyGroupForRole(group, role) {
				continue
			}
			enableGroups = append(enableGroups, group)
		}
		item.EnableGroup = enableGroups
		filtered = append(filtered, item)
	}
	return filtered
}

func GetPricing(c *gin.Context) {
	pricing := model.GetPricing()
	userId, exists := c.Get("id")
	usableGroup := map[string]string{}
	groupRatio := map[string]float64{}
	for s, f := range ratio_setting.GetGroupRatioCopy() {
		groupRatio[s] = f
	}
	var group string
	// HeaderNavModuleAuth("pricing") falls through to TryUserAuth for a public
	// module, which only writes the auth context when the credential is valid.
	// An anonymous request therefore has no role on the context and resolves as
	// RoleGuestUser (0), pruning every admin-only group from all four surfaces.
	role := common.GetContextKeyInt(c, constant.ContextKeyUserRole)
	if exists {
		user, err := model.GetUserCache(userId.(int))
		if err == nil {
			group = user.Group
			for g := range groupRatio {
				ratio, ok := ratio_setting.GetGroupGroupRatio(group, g)
				if ok {
					groupRatio[g] = ratio
				}
			}
		}
	}

	usableGroup = service.GetUserUsableGroups(group, role)
	pricing = filterPricingByUsableGroups(pricing, usableGroup, role)
	// check groupRatio contains usableGroup
	for group := range ratio_setting.GetGroupRatioCopy() {
		if _, ok := usableGroup[group]; !ok {
			delete(groupRatio, group)
		}
	}

	c.JSON(200, gin.H{
		"success":            true,
		"data":               pricing,
		"vendors":            model.GetVendors(),
		"group_ratio":        groupRatio,
		"usable_group":       usableGroup,
		"supported_endpoint": model.GetSupportedEndpointMap(),
		"auto_groups":        service.GetUserAutoGroup(group, role),
		"pricing_version":    "a42d372ccf0b5dd13ecf71203521f9d2",
	})
}

func ResetModelRatio(c *gin.Context) {
	defaultStr := ratio_setting.DefaultModelRatio2JSONString()
	err := model.UpdateOption("ModelRatio", defaultStr)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	err = ratio_setting.UpdateModelRatioByJSONString(defaultStr)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"message": "重置模型倍率成功",
	})
}
