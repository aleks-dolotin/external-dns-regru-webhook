Feature: Reg Adapter high-risk behaviors

  Scenario: Respect Retry-After header and backoff
    Given mock Reg.ru returns HTTP 429 with header "Retry-After: 5"
    When the adapter issues a request that receives 429
    Then the adapter waits at least 5 seconds before retrying
    And the retry is scheduled with exponential backoff and jitter

  Scenario: Reapplying manifest does not create duplicates
    Given a K8s Ingress is applied producing a create operation
    When the same manifest is applied again
    Then no additional DNS record is created

  Scenario: Credential rotation
    Given adapter uses secret "reg-credentials-v1"
    When operator rotates to "reg-credentials-v2" and swaps references
    Then adapter uses new credentials without full restart

  Scenario: Reconciliation repairs drift
    Given Reg.ru is missing a record present in K8s
    When operator triggers force-resync
    Then the adapter creates the missing record in Reg.ru

