package response

import "time"

type CreateAgentConfigResp struct {
	TemplateID   string    `json:"template_id"`
	OpenFileName string    `json:"open_file_name"`
	CreatedAt    time.Time `json:"created_at"`
}
