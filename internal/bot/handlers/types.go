package handlers

import (
	"github.com/vladimiradmaev/diabetes-helper/internal/interfaces"
)

// Dependencies holds all service dependencies for handlers
type Dependencies struct {
	UserService     interfaces.UserServiceInterface
	FoodAnalysisSvc interfaces.FoodAnalysisServiceInterface
	BloodSugarSvc   interfaces.BloodSugarServiceInterface
	InsulinSvc      interfaces.InsulinServiceInterface
}
