package handshake_test

import (
	"errors"
	"fmt"
	"testing"

	"util.tim/encrypto/core/asymetric"
	"util.tim/encrypto/core/communication/handshake"
	"util.tim/encrypto/core/subscribable"
)

func TestHandler_SendVerification_ShouldInitiallySendAnError(t *testing.T) {
	testHelper := newTestHelper(t)
	failureCalled := false
	handler := handshake.NewHandler(
		&testResponder{
			onPublicKey: func(i []byte) {
				t.Log("No success response should be sent")
				t.Fail()
			},
			onFailure: func(message string) {
				failureCalled = true

				expectedFailureMessage := "Error: Can not send a verification code without the client public key"
				testHelper.ExpectStringsToMatch(expectedFailureMessage, message)
			},
		},
		newTestProvider(func(tpp *TestProviderProps) {}),
	)

	handler.SendVerification("fancy code")

	if failureCalled == false {
		t.Log("Expected ErrorResponse to have been called")
		t.Fail()
	}
}

func TestHandler_SendVerification_successful_ReceiveKey_sends_KeyRecievedResponse(t *testing.T) {
	encryptionPrefix := "Encrypted"
	onKeyReceivedCalled := false
	handler := handshake.NewHandler(
		&testResponder{
			onPublicKey: func(key []byte) {},
			onKeyReceived: func() {
				onKeyReceivedCalled = true
			},
			onFailure: func(message string) {
				t.Log("ErrorResponse should not be called")
				t.Fail()
			},
		},
		newTestProvider(func(tpp *TestProviderProps) {
			tpp.GetClientContainerResult = func() (asymetric.RemoteRSAContainer, error) {
				return newSuccessRemoteRSAContainer(encryptionPrefix), nil
			}
		}),
	)

	handler.ReceiveKey([]byte("A public key"))

	if onKeyReceivedCalled == false {
		t.Log("Expected KeyReceived to have been called")
		t.Fail()
	}
}

func TestHandler_SendVerification_after_successful_ReceiveKey_when_encryptionSuccessful_callsVerificationResponse(t *testing.T) {
	testHelper := newTestHelper(t)
	verificationCode := "fancy code"
	encryptionPrefix := "Encrypted"
	sendVerificationCalled := false
	handler := handshake.NewHandler(
		&testResponder{
			onPublicKey:   func(key []byte) {},
			onKeyReceived: func() {},
			onFailure: func(message string) {
				t.Log("ErrorResponse should not be called")
				t.Fail()
			},
			onSendVerify: func(verification []byte) {
				sendVerificationCalled = true

				expectedEncryptedCode := fmt.Sprintf("%s: [%s]", encryptionPrefix, verificationCode)
				actualEncryptedCodeString := string(verification)

				testHelper.ExpectStringsToMatch(expectedEncryptedCode, actualEncryptedCodeString)
			},
		},
		newTestProvider(func(tpp *TestProviderProps) {
			tpp.GetClientContainerResult = func() (asymetric.RemoteRSAContainer, error) {
				return newSuccessRemoteRSAContainer(encryptionPrefix), nil
			}
		}),
	)

	handler.ReceiveKey([]byte("A public key"))
	handler.SendVerification(verificationCode)

	if sendVerificationCalled == false {
		t.Log("Expected SendVerification to have been called")
		t.Fail()
	}
}

func TestHandler_SendVerification_after_successful_ReceiveKey_when_encryptionError_callsFailureResponse(t *testing.T) {
	testHelper := newTestHelper(t)
	verificationCode := "fancy code"
	expectedFailureMessage := "Encrypted"
	failureResponseCalled := false
	handler := handshake.NewHandler(
		&testResponder{
			onPublicKey:   func(key []byte) {},
			onKeyReceived: func() {},
			onFailure: func(message string) {
				failureResponseCalled = true
				testHelper.ExpectStringsToMatch(expectedFailureMessage, message)
			},
			onSendVerify: func(verification []byte) {
				t.Log("VerificationResponse should not be called")
				t.Fail()
			},
		},
		newTestProvider(func(tpp *TestProviderProps) {
			tpp.GetClientContainerResult = func() (asymetric.RemoteRSAContainer, error) {
				return newFailureRemoteRSAContainer(expectedFailureMessage), nil
			}
		}),
	)

	handler.ReceiveKey([]byte("A public key"))
	handler.SendVerification(verificationCode)

	if failureResponseCalled == false {
		t.Log("Expected ErrorResponse to have been called")
		t.Fail()
	}
}

func TestHandler_ReceivePrivateKey_errorCreatingRemoteRSAContainer_callsFailureResponse(t *testing.T) {
	testHelper := newTestHelper(t)
	expectedFailureMessage := "failed to create RemoteRSAContainer"
	failureResponseCalled := false
	handler := handshake.NewHandler(
		&testResponder{
			onPublicKey:   func(key []byte) {},
			onKeyReceived: func() {},
			onFailure: func(message string) {
				failureResponseCalled = true
				testHelper.ExpectStringsToMatch(expectedFailureMessage, message)
			},
			onSendVerify: func(verification []byte) {
				t.Log("Verificaiton should not be sent")
				t.Fail()
			},
		},
		newTestProvider(func(tpp *TestProviderProps) {
			tpp.GetClientContainerResult = func() (asymetric.RemoteRSAContainer, error) {
				return nil, errors.New(expectedFailureMessage)
			}
		}),
	)

	handler.ReceiveKey([]byte("A public key"))

	if failureResponseCalled == false {
		t.Log("Expected FailureResponse to have been called")
		t.Fail()
	}
}

func TestHandler_after_ReceivePrivateKey_errorCreatingRemoteRSAContainer_SendVerification_callsFailureResponse(t *testing.T) {
	testHelper := newTestHelper(t)
	expectedFailureMessage := "failed to create RemoteRSAContainer"
	failureResponses := []string{}
	handler := handshake.NewHandler(
		&testResponder{
			onPublicKey:   func(key []byte) {},
			onKeyReceived: func() {},
			onFailure: func(message string) {
				failureResponses = append(failureResponses, message)
			},
			onSendVerify: func(verification []byte) {
				t.Log("Verificaiton should not be sent")
				t.Fail()
			},
		},
		newTestProvider(func(tpp *TestProviderProps) {
			tpp.GetClientContainerResult = func() (asymetric.RemoteRSAContainer, error) {
				return nil, errors.New(expectedFailureMessage)
			}
		}),
	)

	handler.ReceiveKey([]byte("A public key"))
	handler.SendVerification("Secret Code")

	failureResponseCount := len(failureResponses)
	if failureResponseCount < 2 {
		t.Log("Expected FailureResponse to have been called at least twice")
		t.Fail()
	}

	lastFailureResponse := failureResponses[failureResponseCount-1]
	testHelper.ExpectStringsToMatch("Error: Can not send a verification code without the client public key", lastFailureResponse)
}

func TestHandler_SendPublicKey_ShouldSendPublicKey(t *testing.T) {
	testHelper := newTestHelper(t)

	publicKeyResponseCalled := false
	expectedKeyString := "A Public Key from the server"
	handler := handshake.NewHandler(
		&testResponder{
			onPublicKey: func(key []byte) {
				publicKeyResponseCalled = true
				actualKeyString := string(key)
				testHelper.ExpectStringsToMatch(expectedKeyString, actualKeyString)
			},
			onFailure: func(message string) {
				t.Log("ErrorResponse should not be called")
				t.Fail()
			},
		},
		newTestProvider(func(tpp *TestProviderProps) {
			tpp.GetServerContainer = func() asymetric.LocalRSAContainer {
				return newLocalRSAContainer(func(props *TestLocalRSAProps) {
					props.PublicKey = expectedKeyString
				})
			}
		}),
	)

	handler.SendKey()

	if publicKeyResponseCalled == false {
		t.Log("Expected PublicKeyResponse to be called")
		t.Fail()
	}
}

func TestHandler_Verify_signalsAnError(t *testing.T) {
	helper := newTestHelper(t)

	expectedFailureMessage := "Verification not possible before keys have been exchanged"
	failureResponseCalled := false
	handler := handshake.NewHandler(
		&testResponder{
			onFailure: func(actualFailureMessage string) {
				failureResponseCalled = true
				helper.ExpectStringsToMatch(expectedFailureMessage, actualFailureMessage)
			},
		},
		newTestProvider(func(tpp *TestProviderProps) {}),
	)

	handler.Verify([]byte("some code"))

	if failureResponseCalled == false {
		t.Log("FailureResponse should have been called")
		t.Fail()
	}
}

func TestHandler_Verify_after_SendPublicKey(t *testing.T) {
	helper := newTestHelper(t)

	expectedFailureMessage := "Verification not possible before client key received"
	failureResponseCalled := false
	handler := handshake.NewHandler(
		&testResponder{
			onFailure: func(actualFailureMessage string) {
				failureResponseCalled = true
				helper.ExpectStringsToMatch(expectedFailureMessage, actualFailureMessage)
			},
			onPublicKey: func(b []byte) {},
		},
		newTestProvider(func(tpp *TestProviderProps) {}),
	)

	handler.SendKey()
	handler.Verify([]byte("some code"))

	if failureResponseCalled == false {
		t.Log("FailureResponse should have been called")
		t.Fail()
	}
}

func TestHandler_Verify_after_successful_ReceivePublicKey(t *testing.T) {
	helper := newTestHelper(t)

	expectedFailureMessage := "Verification not possible before server key sent"
	failureResponseCalled := false
	handler := handshake.NewHandler(
		&testResponder{
			onFailure: func(actualFailureMessage string) {
				failureResponseCalled = true
				helper.ExpectStringsToMatch(expectedFailureMessage, actualFailureMessage)
			},
			onKeyReceived: func() {},
			onPublicKey:   func(b []byte) {},
		},
		newTestProvider(func(tpp *TestProviderProps) {
			tpp.GetClientContainerResult = func() (asymetric.RemoteRSAContainer, error) {
				return newSuccessRemoteRSAContainer("Prefix"), nil
			}
		}),
	)

	handler.ReceiveKey([]byte{})
	handler.Verify([]byte("some code"))

	if failureResponseCalled == false {
		t.Log("FailureResponse should have been called")
		t.Fail()
	}
}

func TestHandler_Verify_after_ReceivePublicKey_error(t *testing.T) {
	helper := newTestHelper(t)

	expectedFailureMessage := "Verification not possible before keys have been exchanged"
	failureMessages := []string{}
	handler := handshake.NewHandler(
		&testResponder{
			onFailure: func(actualFailureMessage string) {
				failureMessages = append(failureMessages, actualFailureMessage)
			},
			onPublicKey: func(b []byte) {},
		},
		newTestProvider(func(tpp *TestProviderProps) {
			tpp.GetClientContainerResult = func() (asymetric.RemoteRSAContainer, error) {
				return nil, errors.New("some error")
			}
		}),
	)

	handler.ReceiveKey([]byte{})
	handler.Verify([]byte("some code"))

	failureMessageCount := len(failureMessages)
	if failureMessageCount < 2 {
		t.Log("FailureResponse should have been called at least twice")
		t.Fail()
	}

	actualFailureMessage := failureMessages[failureMessageCount-1]
	helper.ExpectStringsToMatch(expectedFailureMessage, actualFailureMessage)
}

func TestHandler_Verify_after_ReceivePublicKey_and_SendPublicKey(t *testing.T) {
	helper := newTestHelper(t)

	expectedFailureMessage := "Verification not possible before verification message sent"
	failureMessages := []string{}
	handler := handshake.NewHandler(
		&testResponder{
			onFailure: func(actualFailureMessage string) {
				failureMessages = append(failureMessages, actualFailureMessage)
			},
			onPublicKey:   func(b []byte) {},
			onKeyReceived: func() {},
		},
		newTestProvider(func(tpp *TestProviderProps) {
			tpp.GetClientContainerResult = func() (asymetric.RemoteRSAContainer, error) {
				return newSuccessRemoteRSAContainer("something"), nil
			}
		}),
	)

	handler.ReceiveKey([]byte{})
	handler.SendKey()
	handler.Verify([]byte("some code"))

	failureMessageCount := len(failureMessages)
	if failureMessageCount < 1 {
		t.Log("FailureResponse should have been called at least once")
		t.Fail()
	}

	actualFailureMessage := failureMessages[failureMessageCount-1]
	helper.ExpectStringsToMatch(expectedFailureMessage, actualFailureMessage)
}

func TestHandler_Verify_after_ReceivePublicKey_SendPublicKey_SendVerification_and_correctVerificationCode_signalsVerification(t *testing.T) {
	expectedVerification := "expectedVerification"
	onVerifiedCalled := false
	failureMessages := []string{}
	handler := handshake.NewHandler(
		&testResponder{
			onFailure: func(actualFailureMessage string) {
				failureMessages = append(failureMessages, actualFailureMessage)
			},
			onPublicKey:   func(b []byte) {},
			onKeyReceived: func() {},
			onSendVerify:  func(b []byte) {},
			onVerified: func(c subscribable.Connection, rr asymetric.RemoteRSAContainer, lr asymetric.LocalRSAContainer) {
				onVerifiedCalled = true
			},
		},
		newTestProvider(func(tpp *TestProviderProps) {
			tpp.GetServerContainer = func() asymetric.LocalRSAContainer {
				return newLocalRSAContainer(func(props *TestLocalRSAProps) {
					props.DecryptResult = func(dh asymetric.DecryptHandler, b []byte) {
						dh.Success(expectedVerification)
					}
				})
			}
		}),
	)

	handler.ReceiveKey([]byte{})
	handler.SendKey()
	handler.SendVerification(expectedVerification)

	handler.Verify([]byte(expectedVerification))

	if len(failureMessages) > 0 {
		t.Log("There should be no failure calls")
		t.Fail()
	}
	if onVerifiedCalled == false {
		t.Log("OnVerified should have been called")
		t.Fail()
	}
}

func TestHandler_Verify_after_ReceivePublicKey_SendPublicKey_SendVerification_and_wrongVerificationMessage_signalsFailure(t *testing.T) {
	helper := newTestHelper(t)

	onVerifiedCalled := false
	failureMessages := []string{}
	handler := handshake.NewHandler(
		&testResponder{
			onFailure: func(actualFailureMessage string) {
				failureMessages = append(failureMessages, actualFailureMessage)
			},
			onPublicKey:   func(b []byte) {},
			onKeyReceived: func() {},
			onSendVerify:  func(b []byte) {},
			onVerified: func(c subscribable.Connection, rr asymetric.RemoteRSAContainer, lr asymetric.LocalRSAContainer) {
				onVerifiedCalled = true
			},
		},
		newTestProvider(func(tpp *TestProviderProps) {
			tpp.GetServerContainer = func() asymetric.LocalRSAContainer {
				return newLocalRSAContainer(func(props *TestLocalRSAProps) {
					props.DecryptResult = func(dh asymetric.DecryptHandler, b []byte) {
						dh.Success("Does not match")
					}
				})
			}
		}),
	)

	handler.ReceiveKey([]byte{})
	handler.SendKey()
	handler.SendVerification("Verification code")

	handler.Verify([]byte("Wrong_prefix: Verification code"))

	failureMessageCount := len(failureMessages)
	if failureMessageCount < 1 {
		t.Log("There should be at least one failure message")
		t.Fail()
	}

	lastFailureMessage := failureMessages[failureMessageCount-1]
	helper.ExpectStringsToMatch("verification failed", lastFailureMessage)

	if onVerifiedCalled == true {
		t.Log("OnVerified should not have been called")
		t.Fail()
	}
}
