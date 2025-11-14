package whatsapp

// WhatsApp Message Templates
// These templates must be pre-approved by WhatsApp Business

const (
	// TemplateLoyaltyWelcome is sent when a customer enrolls
	TemplateLoyaltyWelcome = "loyalty_welcome"

	// TemplateRewardIssued is sent when a reward is issued
	TemplateRewardIssued = "reward_issued"

	// TemplateRewardReminder is sent before a reward expires
	TemplateRewardReminder = "reward_reminder"

	// TemplateRewardRedeemed is sent when a reward is redeemed
	TemplateRewardRedeemed = "reward_redeemed"
)

// Template definitions with parameter descriptions
// These are for reference - actual templates are configured in WhatsApp Business Manager

// LOYALTY_WELCOME
// Parameters:
// 1. {{1}} - Customer name or phone number
// Example: "Welcome to our loyalty program, {{1}}! You can now earn rewards on every purchase."

// REWARD_ISSUED
// Parameters:
// 1. {{1}} - Reward name
// 2. {{2}} - Reward code
// 3. {{3}} - Expiry date (if applicable)
// Example: "Congratulations! You've earned {{1}}. Code: {{2}}. Valid until {{3}}."

// REWARD_REMINDER
// Parameters:
// 1. {{1}} - Reward name
// 2. {{2}} - Reward code
// 3. {{3}} - Days until expiry
// Example: "Reminder: Your {{1}} (Code: {{2}}) expires in {{3}} days. Use it soon!"

// REWARD_REDEEMED
// Parameters:
// 1. {{1}} - Reward name
// 2. {{2}} - Redemption location/date
// Example: "Your {{1}} has been successfully redeemed at {{2}}. Thank you!"

// FormatWelcomeParams creates parameters for the welcome template
func FormatWelcomeParams(customerName string) map[string]string {
	return map[string]string{
		"1": customerName,
	}
}

// FormatRewardIssuedParams creates parameters for the reward issued template
func FormatRewardIssuedParams(rewardName, code, expiry string) map[string]string {
	return map[string]string{
		"1": rewardName,
		"2": code,
		"3": expiry,
	}
}

// FormatRewardReminderParams creates parameters for the reward reminder template
func FormatRewardReminderParams(rewardName, code, daysUntilExpiry string) map[string]string {
	return map[string]string{
		"1": rewardName,
		"2": code,
		"3": daysUntilExpiry,
	}
}

// FormatRewardRedeemedParams creates parameters for the reward redeemed template
func FormatRewardRedeemedParams(rewardName, location string) map[string]string {
	return map[string]string{
		"1": rewardName,
		"2": location,
	}
}

// Help messages (sent as regular text messages)

const (
	HelpMessage = `*Zimbabwe Loyalty Program Help*

Available commands:
• /enroll - Join the loyalty program
• /balance - Check your points balance
• /rewards - View available rewards
• /myrewards - See your active rewards
• /redeem [code] - Redeem a reward
• /refer - Get your referral link
• /help - Show this help message

Simply send a command to get started!`

	WelcomeMessage = `Welcome to the Zimbabwe Loyalty Program! 🎉

You've successfully enrolled. Start earning rewards with every purchase!

Send /help to see what you can do.`

	EnrollmentCompleteMessage = `Thank you for enrolling!

Your loyalty account is now active. You'll receive notifications about new rewards and special offers.

Use /balance to check your points.`

	NoRewardsMessage = `You don't have any active rewards yet.

Keep shopping and earning points to unlock exciting rewards!

Send /rewards to see what's available.`

	InvalidCommandMessage = `I didn't understand that command.

Send /help to see available commands.`

	ErrorMessage = `Sorry, something went wrong. Please try again later or contact support.`
)
