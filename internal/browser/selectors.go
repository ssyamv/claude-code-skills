package browser

type Selectors struct {
	CreateButton         string
	AppIDValue           string
	AppSecretReveal      string
	CallbackInput        string
	ScopeSearchInput     string
	PublishButton        string
	OpenPermissionButton string
	CreateVersionButton  string
	SafeAddButton        string
	SafeBatchEditButton  string
	SafeValueText        string
	ScopeTable           string
	ScopeNameText        string
	VersionTable         string
}

func DefaultSelectors(callbackURL string) Selectors {
	return Selectors{
		CreateButton:         `button.data-test__create-app-button`,
		AppIDValue:           `.auth-info__appid .ud__form__item__control__input__content, .auth-info__wrapper`,
		AppSecretReveal:      `button[data-testid="reveal-secret"]`,
		CallbackInput:        `input[value="` + callbackURL + `"], textarea`,
		ScopeSearchInput:     `input[placeholder*="搜索"], input[placeholder*="scope"]`,
		PublishButton:        `button.ud__button--filled.ud__button--filled-default.ud__button--size-md`,
		OpenPermissionButton: `button.ud__button--text.ud__button--text-primary.ud__button--size-md`,
		CreateVersionButton:  `button.ud__button--filled.ud__button--filled-default.ud__button--size-md`,
		SafeAddButton:        `button.ud__button--filled.ud__button--filled-default.ud__button--size-md`,
		SafeBatchEditButton:  `button.safe-setting-item__batch-edit-button`,
		SafeValueText:        `.safe-item__item-value`,
		ScopeTable:           `.scope-info-table`,
		ScopeNameText:        `.scope-info-table__info__name`,
		VersionTable:         `.version-list__table`,
	}
}
