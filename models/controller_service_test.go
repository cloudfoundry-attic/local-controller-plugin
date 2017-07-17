package models_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/jeffpak/csi"
	"golang.org/x/net/context"
	"time"
	"code.cloudfoundry.org/goshims/osshim/os_fake"
	"code.cloudfoundry.org/goshims/filepathshim/filepath_fake"
	"github.com/jeffpak/local-controller-plugin/models"
)

var _ = Describe("ControllerService", func() {
	var(
		cs *models.Controller
		context context.Context

		fakeOs       *os_fake.FakeOs
		fakeFilepath *filepath_fake.FakeFilepath
		mountDir     string
		volumeName   string
		volID        *VolumeID
		vc           []*VolumeCapability
		volInfo			 *VolumeInfo
		err          error
	)

	BeforeEach(func() {
		mountDir = "/path/to/mount"
		fakeOs = &os_fake.FakeOs{}
		fakeFilepath = &filepath_fake.FakeFilepath{}
		cs = models.NewController(fakeOs, fakeFilepath, mountDir)
		context = &DummyContext{}
		volID = &VolumeID{Values: map[string]string{"volume_name": "abcd"}}
		volumeName = "abcd"
		vc = []*VolumeCapability{{Value: &VolumeCapability_Mount{}}}
		volInfo	 = &VolumeInfo{
							AccessMode: &AccessMode{Mode:AccessMode_UNKNOWN},
							Id: volID}
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
		})

		Describe("ControllerPublishVolume", func() {
			var (
				request *ControllerPublishVolumeRequest
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
				request *ControllerUnpublishVolumeRequest
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
				request *ValidateVolumeCapabilitiesRequest
				expectedResponse *ValidateVolumeCapabilitiesResponse
			)
			Context("when ValidateVolumeCapabilities is called with a ValidateVolumeCapabilitiesRequest", func() {
				BeforeEach(func() {
					request = &ValidateVolumeCapabilitiesRequest{
						&Version{Major: 0, Minor: 0, Patch: 1},
						volInfo,
						[]*VolumeCapability{&VolumeCapability{}},
					}
				})
				JustBeforeEach(func() {
					expectedResponse, err = cs.ValidateVolumeCapabilities(context, request)
				})
				It("should return a ValidateVolumeResponse", func() {
					Expect(*expectedResponse).NotTo(BeNil())
				})
			})
		})


		Describe("ListVolumes", func() {
			var (
				request *ListVolumesRequest
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
				request *GetCapacityRequest
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
				request *ControllerGetCapabilitiesRequest
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
				It("should return a ControllerGetCapabilitiesResponse", func() {
					Expect(*expectedResponse).NotTo(BeNil())
				})
			})
		})
	})

	Describe("DeleteVolume", func() {
		var (
			response *DeleteVolumeResponse
		)

		It("should fail if no volume name is provided", func() {
			_, err := cs.DeleteVolume(context, &DeleteVolumeRequest{
				VolumeId: &VolumeID{},
			})
			Expect(err.Error()).To(Equal("Missing mandatory 'volume_name'"))
		})

		It("should fail if no volume was created", func() {
			volID = &VolumeID{Values: map[string]string{"volume_name": volumeName}}
			_, err := cs.DeleteVolume(context, &DeleteVolumeRequest{
				VolumeId: volID,
			})
			Expect(err.Error()).To(Equal("Volume '" + volumeName + "' not found"))
		})

		Context("when the volume has been created", func() {
			BeforeEach(func() {
				createSuccessful(context, cs, fakeOs, volumeName, vc)
			})

			It("destroys the volume", func() {
				volID = &VolumeID{Values: map[string]string{"volume_name": volumeName}}
				deleteSuccessful(context, cs, *volID)
				Expect(fakeOs.RemoveAllCallCount()).To(Equal(1))

				_, err = cs.DeleteVolume(context, &DeleteVolumeRequest{
					VolumeId: volID,
				})
				Expect(err.Error()).To(Equal("Volume '" + volumeName + "' not found"))
			})
		})

		Context("when the volume has not been created", func() {
			It("returns an error", func() {
				response, err = cs.DeleteVolume(context, &DeleteVolumeRequest{
					VolumeId: volID,
				})
				Expect(err.Error()).To(Equal("Volume '" + volumeName + "' not found"))
			})
		})
	})
})


type DummyContext struct {}

func (*DummyContext) Deadline() (deadline time.Time, ok bool) { return time.Time{}, false }

func (*DummyContext) Done() <-chan struct{} {return nil}

func (*DummyContext) Err() (error){ return nil }

func (*DummyContext) Value(key interface{}) interface{} {return nil}

func createSuccessful(ctx context.Context, cs ControllerServer, fakeOs *os_fake.FakeOs, volumeName string, vc []*VolumeCapability) *CreateVolumeResponse {
	createResponse, err := cs.CreateVolume(ctx, &CreateVolumeRequest{
		Version: &Version{},
		Name: volumeName,
		VolumeCapabilities: vc,
	})
	Expect(err).To(BeNil())
	return createResponse
}

func deleteSuccessful(ctx context.Context, cs ControllerServer, volumeID VolumeID) *DeleteVolumeResponse{
	deleteResponse, err := cs.DeleteVolume(ctx, &DeleteVolumeRequest{
		Version: &Version{},
		VolumeId: &volumeID,
	})
	Expect(err).To(BeNil())
	return deleteResponse
}
