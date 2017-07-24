package controller_test

import (
	"time"

	"code.cloudfoundry.org/goshims/filepathshim/filepath_fake"
	"code.cloudfoundry.org/goshims/osshim/os_fake"
	. "github.com/jeffpak/csi"
	"github.com/jeffpak/local-controller-plugin/controller"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"
)

var _ = Describe("ControllerService", func() {
	var (
		cs      *controller.Controller
		context context.Context

		fakeOs       *os_fake.FakeOs
		fakeFilepath *filepath_fake.FakeFilepath
		mountDir     string
		volumeName   string
		volID        *VolumeID
		vc           []*VolumeCapability
		volInfo      *VolumeInfo
		err          error
	)

	BeforeEach(func() {
		mountDir = "/path/to/mount"
		fakeOs = &os_fake.FakeOs{}
		fakeFilepath = &filepath_fake.FakeFilepath{}
		cs = controller.NewController(fakeOs, fakeFilepath, mountDir)
		context = &DummyContext{}
		volID = &VolumeID{Values: map[string]string{"volume_name": "vol-name"}}
		volumeName = "vol-name"
		vc = []*VolumeCapability{{Value: &VolumeCapability_Mount{Mount: &VolumeCapability_MountVolume{}}}}
		volInfo = &VolumeInfo{
			AccessMode: &AccessMode{Mode: AccessMode_UNKNOWN},
			Id:         volID}
	})

	Describe("CreateVolume", func() {
		var (
			expectedResponse *CreateVolumeResponse
		)

		Context("when CreateVolume is called with a CreateVolumeRequest", func() {
			BeforeEach(func() {
				expectedResponse = createSuccessful(context, cs, fakeOs, volumeName, vc)
			})

			It("does not fail", func() {
				Expect(*expectedResponse).To(Equal(CreateVolumeResponse{
					Reply: &CreateVolumeResponse_Result_{
						Result: &CreateVolumeResponse_Result{
							VolumeInfo: volInfo,
						},
					},
				}))
			})

			Context("when the Volume exists", func() {
				BeforeEach(func() {
					expectedResponse = createSuccessful(context, cs, fakeOs, volumeName, vc)
				})

				It("should succeed and respond with the existent volume", func() {
					Expect(*expectedResponse).To(Equal(CreateVolumeResponse{
						Reply: &CreateVolumeResponse_Result_{
							Result: &CreateVolumeResponse_Result{
								VolumeInfo: volInfo,
							},
						},
					}))
				})
			})
		})

		Describe("DeleteVolume", func() {
			It("should fail if no volume ID is provided in the request", func() {
				_, err = cs.DeleteVolume(context, &DeleteVolumeRequest{})
				Expect(err.Error()).To(Equal("Request missing 'volume_name'"))
			})

			It("should fail if volume name is empty", func() {
				_, err = cs.DeleteVolume(context, &DeleteVolumeRequest{
					VolumeId: &VolumeID{Values: map[string]string{"volume_name": ""}},
				})
				Expect(err.Error()).To(Equal("Request needs non-empty 'volume_name'"))
			})

			It("should fail if no volume was found", func() {
				_, err = cs.DeleteVolume(context, &DeleteVolumeRequest{
					VolumeId: &VolumeID{Values: map[string]string{"volume_name": volumeName}},
				})
				Expect(err.Error()).To(Equal("Volume '" + volumeName + "' not found"))
			})

			Context("when the volume has been created", func() {
				BeforeEach(func() {
					createSuccessful(context, cs, fakeOs, volumeName, vc)
				})

				It("does not fail", func() {
					response := deleteSuccessful(context, cs, volID)
					Expect(response).To(Equal(&DeleteVolumeResponse{
						Reply: &DeleteVolumeResponse_Result_{
							Result: &DeleteVolumeResponse_Result{},
						},
					}))
				})
			})
		})

		Describe("ControllerPublishVolume", func() {
			var (
				request          *ControllerPublishVolumeRequest
				expectedResponse *ControllerPublishVolumeResponse
			)

			Context("when ControllerPublishVolume is called with a ControllerPublishVolumeRequest", func() {
				BeforeEach(func() {
					request = &ControllerPublishVolumeRequest{
						&Version{Major: 0, Minor: 0, Patch: 1},
						volID,
						&VolumeMetadata{Values: map[string]string{}},
						&NodeID{Values: map[string]string{}},
						false,
					}
				})
				JustBeforeEach(func() {
					expectedResponse, err = cs.ControllerPublishVolume(context, request)
				})
				It("should return a ControllerPublishVolumeResponse", func() {
					Expect(*expectedResponse).NotTo(BeNil())
				})
			})
		})

		Describe("ControllerUnpublishVolume", func() {
			var (
				request          *ControllerUnpublishVolumeRequest
				expectedResponse *ControllerUnpublishVolumeResponse
			)
			Context("when ControllerUnpublishVolume is called with a ControllerUnpublishVolumeRequest", func() {
				BeforeEach(func() {
					request = &ControllerUnpublishVolumeRequest{
						&Version{Major: 0, Minor: 0, Patch: 1},
						volID,
						&VolumeMetadata{Values: map[string]string{}},
						&NodeID{Values: map[string]string{}},
					}
				})
				JustBeforeEach(func() {
					expectedResponse, err = cs.ControllerUnpublishVolume(context, request)
				})
				It("should return a ControllerUnpublishVolumeResponse", func() {
					Expect(*expectedResponse).NotTo(BeNil())
				})
			})
		})

		Describe("ValidateVolumeCapabilities", func() {
			var (
				request          *ValidateVolumeCapabilitiesRequest
				expectedResponse *ValidateVolumeCapabilitiesResponse
			)
			Context("when called with no capabilities", func() {
				BeforeEach(func() {
					request = &ValidateVolumeCapabilitiesRequest{
						&Version{Major: 0, Minor: 0, Patch: 1},
						volInfo,
						[]*VolumeCapability{{Value: &VolumeCapability_Mount{
							Mount: &VolumeCapability_MountVolume{
								MountFlags: []string{""},
							}}}}}
				})
				JustBeforeEach(func() {
					expectedResponse, err = cs.ValidateVolumeCapabilities(context, request)
				})
				It("should return a ValidateVolumeResponse", func() {
					Expect(err).To(BeNil())
					Expect(expectedResponse).To(Equal(&ValidateVolumeCapabilitiesResponse{
						Reply: &ValidateVolumeCapabilitiesResponse_Result_{
							Result: &ValidateVolumeCapabilitiesResponse_Result{Supported: true},
						}}))
				})
			})

			Context("when called with unsupported FsType capabilities", func() {
				BeforeEach(func() {
					request = &ValidateVolumeCapabilitiesRequest{
						&Version{Major: 0, Minor: 0, Patch: 1},
						volInfo,
						[]*VolumeCapability{{Value: &VolumeCapability_Mount{
							Mount: &VolumeCapability_MountVolume{
								FsType: "unsupported",
							}}}},
					}
				})
				JustBeforeEach(func() {
					expectedResponse, err = cs.ValidateVolumeCapabilities(context, request)
				})
				It("should return an error", func() {
					Expect(err).To(BeNil())
					Expect(expectedResponse).To(Equal(&ValidateVolumeCapabilitiesResponse{
						Reply: &ValidateVolumeCapabilitiesResponse_Result_{
							Result: &ValidateVolumeCapabilitiesResponse_Result{
								Supported: false,
								Message:   "Specifying FsType is unsupported.",
							},
						}}))
				})
			})

			Context("when called with unsupported MountFlag capabilities", func() {
				BeforeEach(func() {
					request = &ValidateVolumeCapabilitiesRequest{
						&Version{Major: 0, Minor: 0, Patch: 1},
						volInfo,
						[]*VolumeCapability{{Value: &VolumeCapability_Mount{
							Mount: &VolumeCapability_MountVolume{
								MountFlags: []string{"unsupported"},
							}}}},
					}
				})
				JustBeforeEach(func() {
					expectedResponse, err = cs.ValidateVolumeCapabilities(context, request)
				})
				It("should return an error", func() {
					Expect(err).To(BeNil())
					Expect(expectedResponse).To(Equal(&ValidateVolumeCapabilitiesResponse{
						Reply: &ValidateVolumeCapabilitiesResponse_Result_{
							Result: &ValidateVolumeCapabilitiesResponse_Result{
								Supported: false,
								Message:   "Specifying mount flags is unsupported.",
							}},
					}))
				})
			})
		})

		Describe("ListVolumes", func() {
			var (
				request          *ListVolumesRequest
				expectedResponse *ListVolumesResponse
			)
			Context("when ListVolumes is called with a ListVolumesRequest", func() {
				BeforeEach(func() {
					request = &ListVolumesRequest{
						&Version{Major: 0, Minor: 0, Patch: 1},
						10,
						"starting-token",
					}
				})
				JustBeforeEach(func() {
					expectedResponse, err = cs.ListVolumes(context, request)
				})
				It("should return a ListVolumesResponse", func() {
					Expect(*expectedResponse).NotTo(BeNil())
				})
			})
		})

		Describe("GetCapacity", func() {
			var (
				request          *GetCapacityRequest
				expectedResponse *GetCapacityResponse
			)
			Context("when GetCapacity is called with a GetCapacityRequest", func() {
				BeforeEach(func() {
					request = &GetCapacityRequest{
						&Version{Major: 0, Minor: 0, Patch: 1},
					}
				})
				JustBeforeEach(func() {
					expectedResponse, err = cs.GetCapacity(context, request)
				})
				It("should return a GetCapacityResponse", func() {
					Expect(*expectedResponse).NotTo(BeNil())
				})
			})
		})

		Describe("ControllerGetCapabilities", func() {
			var (
				request          *ControllerGetCapabilitiesRequest
				expectedResponse *ControllerGetCapabilitiesResponse
			)
			Context("when ControllerGetCapabilities is called with a ControllerGetCapabilitiesRequest", func() {
				BeforeEach(func() {
					request = &ControllerGetCapabilitiesRequest{
						&Version{Major: 0, Minor: 0, Patch: 1},
					}
				})
				JustBeforeEach(func() {
					expectedResponse, err = cs.ControllerGetCapabilities(context, request)
				})

				It("should return a ControllerGetCapabilitiesResponse with only CREATE_DELETE_VOLUME specified", func() {
					Expect(expectedResponse).NotTo(BeNil())
					capabilities := expectedResponse.GetResult().GetCapabilities()
					Expect(capabilities).To(HaveLen(1))
					Expect(capabilities[0].GetRpc().GetType()).To(Equal(ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME))
				})
			})
		})
	})
})

type DummyContext struct{}

func (*DummyContext) Deadline() (deadline time.Time, ok bool) { return time.Time{}, false }

func (*DummyContext) Done() <-chan struct{} { return nil }

func (*DummyContext) Err() error { return nil }

func (*DummyContext) Value(key interface{}) interface{} { return nil }

func createSuccessful(ctx context.Context, cs ControllerServer, fakeOs *os_fake.FakeOs, volumeName string, vc []*VolumeCapability) *CreateVolumeResponse {
	createResponse, err := cs.CreateVolume(ctx, &CreateVolumeRequest{
		Version:            &Version{},
		Name:               volumeName,
		VolumeCapabilities: vc,
	})
	Expect(err).To(BeNil())
	return createResponse
}

func deleteSuccessful(ctx context.Context, cs ControllerServer, volumeID *VolumeID) *DeleteVolumeResponse {
	deleteResponse, err := cs.DeleteVolume(ctx, &DeleteVolumeRequest{
		Version:  &Version{},
		VolumeId: volumeID,
	})
	Expect(err).To(BeNil())
	return deleteResponse
}
