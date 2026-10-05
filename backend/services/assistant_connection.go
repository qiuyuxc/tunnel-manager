package services

import "fmt"

func (client *CloudflareClient) SnapshotForAssistant() (*CloudflareClient, error) {
	if client.store != nil && client.userID != "" {
		if connection, found := client.store.ActiveCFConnection(client.userID); found {
			if !connection.HasToken() || connection.AccountID == "" || client.oauth == nil {
				return nil, fmt.Errorf("当前 Cloudflare 连接不可用")
			}
			token, err := client.oauth.AccessTokenFor(connection)
			if err != nil {
				return nil, err
			}
			return &CloudflareClient{apiToken: token, accountID: connection.AccountID, baseURL: client.baseURL, httpClient: client.httpClient}, nil
		}
	}
	token, err := client.accessToken()
	if err != nil {
		return nil, err
	}
	accountID, err := client.currentAccountID()
	if err != nil {
		return nil, err
	}
	return &CloudflareClient{apiToken: token, accountID: accountID, baseURL: client.baseURL, httpClient: client.httpClient}, nil
}
