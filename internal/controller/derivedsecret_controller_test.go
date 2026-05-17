package controller

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	secretderiverv1alpha1 "github.com/LightJack05/SecretDeriver/api/v1alpha1"
)

var _ = Describe("DerivedSecret Controller", Ordered, func() {
	BeforeAll(func() {
		namespaceA := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace-a",
			},
		}
		Expect(k8sClient.Create(ctx, namespaceA)).To(Succeed())
		namespaceB := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace-b",
			},
		}
		Expect(k8sClient.Create(ctx, namespaceB)).To(Succeed())
	})

	DescribeTable("Given a DerivedSecret resource instance",
		func(parentSecret *corev1.Secret, derivedSecret *secretderiverv1alpha1.DerivedSecret, derivedSecretPhase string, derivedSecretConditionReason string, expectedDerivedSecret *corev1.Secret) {
			if parentSecret != nil {
				Expect(k8sClient.Create(ctx, parentSecret)).To(Succeed())
			}
			// Create the DerivedSecret resource
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
			&secretderiverv1alpha1.DerivedSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "derived-secret",
					Namespace: "namespace-a",
				},
				Spec: secretderiverv1alpha1.DerivedSecretSpec{
					ParentSecretRef: v1.SecretReference{
						Name:      "parent-secret",
						Namespace: "namespace-a",
					},
					ParentSecretKey:    "key1",
					GeneratedSecretKey: "key1",
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
		Entry("should fail to create the derived secret when the parent secret is missing",
			nil,
			&secretderiverv1alpha1.DerivedSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "derived-secret",
					Namespace: "namespace-a",
				},
				Spec: secretderiverv1alpha1.DerivedSecretSpec{
					ParentSecretRef: v1.SecretReference{
						Name:      "parent-secret",
						Namespace: "namespace-a",
					},
					ParentSecretKey:    "key1",
					GeneratedSecretKey: "key1",
				},
			},
			"Error",
			"ParentSecretNotFound",
			nil,
		),
		Entry("should fail to create the derived secret when the parent secret key is missing",
			&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "parent-secret",
					Namespace: "namespace-a",
				},
				StringData: map[string]string{
					"key2": "foo",
				},
			},
			&secretderiverv1alpha1.DerivedSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "derived-secret",
					Namespace: "namespace-a",
				},
				Spec: secretderiverv1alpha1.DerivedSecretSpec{
					ParentSecretRef: v1.SecretReference{
						Name:      "parent-secret",
						Namespace: "namespace-a",
					},
					ParentSecretKey:    "key1",
					GeneratedSecretKey: "key1",
				},
			},
			"Error",
			"KeyNotFound",
			nil,
		),
		Entry("should fail to create the derived secret when the parent secret key is empty",
			&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "parent-secret",
					Namespace: "namespace-a",
				},
				StringData: map[string]string{
					"key1": "",
				},
			},
			&secretderiverv1alpha1.DerivedSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "derived-secret",
					Namespace: "namespace-a",
				},
				Spec: secretderiverv1alpha1.DerivedSecretSpec{
					ParentSecretRef: v1.SecretReference{
						Name:      "parent-secret",
						Namespace: "namespace-a",
					},
					ParentSecretKey:    "key1",
					GeneratedSecretKey: "key1",
				},
			},
			"Error",
			"EmptyValue",
			nil,
		),
		Entry("should fail when the parent secret is in a different namespace and the cross namespace access label is not set",
			&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "parent-secret",
					Namespace: "namespace-b",
				},
				StringData: map[string]string{
					"key1": "foo",
				},
			},
			&secretderiverv1alpha1.DerivedSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "derived-secret",
					Namespace: "namespace-a",
				},
				Spec: secretderiverv1alpha1.DerivedSecretSpec{
					ParentSecretRef: v1.SecretReference{
						Name:      "parent-secret",
						Namespace: "namespace-b",
					},
					ParentSecretKey:    "key1",
					GeneratedSecretKey: "key1",
				},
			},
			"Error",
			"ParentSecretNotFound",
			nil,
		),
		Entry("should create the derived secret when the parent secret is in a different namespace but has the secretderiver.lightjack.de/allowCrossnamespaceReference=true label set",
			&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "parent-secret",
					Namespace: "namespace-b",
					Labels: map[string]string{
						"secretderiver.lightjack.de/allowCrossnamespaceReference": "true",
					},
				},
				StringData: map[string]string{
					"key1": "foo",
				},
			},
			&secretderiverv1alpha1.DerivedSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "derived-secret",
					Namespace: "namespace-a",
				},
				Spec: secretderiverv1alpha1.DerivedSecretSpec{
					ParentSecretRef: v1.SecretReference{
						Name:      "parent-secret",
						Namespace: "namespace-b",
					},
					ParentSecretKey:    "key1",
					GeneratedSecretKey: "key1",
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

	AfterEach(func() {
		for _, ns := range []string{"namespace-a", "namespace-b"} {
			derivedSecrets := &secretderiverv1alpha1.DerivedSecretList{}
			Expect(k8sClient.List(ctx, derivedSecrets, client.InNamespace(ns))).To(Succeed())
			for _, ds := range derivedSecrets.Items {
				Expect(k8sClient.Delete(ctx, &ds)).To(Succeed())
			}
			secrets := &corev1.SecretList{}
			Expect(k8sClient.List(ctx, secrets, client.InNamespace(ns))).To(Succeed())
			for _, s := range secrets.Items {
				Expect(k8sClient.Delete(ctx, &s)).To(Succeed())
			}
			Eventually(func() []corev1.Secret {
				leftoverSecrets := &corev1.SecretList{}
				Expect(k8sClient.List(ctx, leftoverSecrets, client.InNamespace(ns))).To(Succeed())
				return leftoverSecrets.Items
			}).WithTimeout(time.Second * 1).Should(BeEmpty())
			Eventually(func() []secretderiverv1alpha1.DerivedSecret {
				leftoverDerivedSecrets := &secretderiverv1alpha1.DerivedSecretList{}
				Expect(k8sClient.List(ctx, leftoverDerivedSecrets, client.InNamespace(ns))).To(Succeed())
				return leftoverDerivedSecrets.Items
			}).WithTimeout(time.Second * 1).Should(BeEmpty())
		}
	})

})
