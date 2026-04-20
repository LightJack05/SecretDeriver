package controller

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	secretderiverv1alpha1 "github.com/LightJack05/SecretDeriver/api/v1alpha1"
)

var _ = Describe("DerivedSecret Controller", func() {
	DescribeTable("Given a DerivedSecret resource instance",
		func(parentSecret *corev1.Secret, derivedSecretPhase string, derivedSecretConditionReason string, expectedDerivedSecret *corev1.Secret) {
			Expect(k8sClient.Create(ctx, parentSecret)).To(Succeed())
			// Create the DerivedSecret resource
			derivedSecret := &secretderiverv1alpha1.DerivedSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      expectedDerivedSecret.Name,
					Namespace: expectedDerivedSecret.Namespace,
				},
				Spec: secretderiverv1alpha1.DerivedSecretSpec{
					ParentSecretRef: v1.SecretReference{
						Name:      parentSecret.Name,
						Namespace: parentSecret.Namespace,
					},
					ParentSecretKey:    "key1",
					GeneratedSecretKey: "key1",
				},
			}
			Expect(k8sClient.Create(ctx, derivedSecret)).To(Succeed())

			// Eventually, the controller should update the DerivedSecret status
			crKey := types.NamespacedName{Name: derivedSecret.Name, Namespace: derivedSecret.Namespace}
			Eventually(func(g Gomega) {
				updatedDerivedSecret := &secretderiverv1alpha1.DerivedSecret{}
				err := k8sClient.Get(ctx, crKey, updatedDerivedSecret)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(updatedDerivedSecret.Status.Phase).To(Equal(derivedSecretPhase))
				g.Expect(updatedDerivedSecret.Status.Conditions).ToNot(BeEmpty())
				g.Expect(updatedDerivedSecret.Status.Conditions[0].Reason).To(Equal(derivedSecretConditionReason))
			}).WithTimeout(time.Second * 10).WithPolling(time.Millisecond * 200).Should(Succeed())

			// If the expected phase is Ready, verify that the derived secret was created with the expected data
			if derivedSecretPhase == "Ready" {
				Eventually(func() (map[string][]byte, error) {
					derivedSecret := &corev1.Secret{}
					err := k8sClient.Get(ctx, types.NamespacedName{Name: expectedDerivedSecret.Name, Namespace: expectedDerivedSecret.Namespace}, derivedSecret)
					return derivedSecret.Data, err
				}).Should(Equal(expectedDerivedSecret.Data))
			}

		},
		Entry("should create the derived secret successfully",
			&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "parent-secret",
					Namespace: "namespace-a",
				},
				StringData: map[string]string{
					"key1": "foo",
				},
			},
			"Ready",
			"DerivationSuccessful",
			&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "derived-secret",
					Namespace: "namespace-a",
				},
				Data: map[string][]byte{
					// Derived key from source material foo with salt 'namespace-a/derived-secret' at 32 bytes, base64 encoded should be ZDkwYWMxNTFlOTdhZTkxNGE1ODk2Y2NmNjBlZTA0NDNhZjJhY2U0MmI2YmJjMGY3NDdkYzJjMjY3ODFiYzA4Yw
					"key1": []byte("GRwhQD1WZM1NvqysEVoTHpq8oH9YV97Fi9VtCURmdHY"),
				},
			},
		),
	)

	BeforeEach(func() {
		namespaceA := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace-a",
			},
		}
		Expect(k8sClient.Create(ctx, namespaceA)).To(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace-a",
			},
		})).To(Succeed())
	})
})
